package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/schollz/progressbar/v3"
)

// ── Config ────────────────────────────────────────────────────────────────────

type Config struct {
	TMDB   TMDBConfig   `toml:"tmdb"`
	Drive  DriveConfig  `toml:"drive"`
	Output OutputConfig `toml:"output"`
	SFTP   SFTPConfig   `toml:"sftp"`
}

type TMDBConfig struct {
	APIKey string `toml:"api_key"`
}

type DriveConfig struct {
	Device string `toml:"device"`
}

type OutputConfig struct {
	Dir string `toml:"dir"`
}

type SFTPConfig struct {
	Host       string `toml:"host"`
	Port       string `toml:"port"`
	User       string `toml:"user"`
	KeyPath    string `toml:"key_path"`
	RemotePath string `toml:"remote_path"`
}

func loadConfig(path string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to load config: %w", err)
	}

	// Allow env var overrides
	if key := os.Getenv("TMDB_API_KEY"); key != "" {
		cfg.TMDB.APIKey = key
	}
	if host := os.Getenv("SFTP_HOST"); host != "" {
		cfg.SFTP.Host = host
	}

	return cfg, nil
}

// ── TMDB ──────────────────────────────────────────────────────────────────────

type TMDBSearchResult struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
}

type TMDBSearchResponse struct {
	Results []TMDBSearchResult `json:"results"`
}

func searchTMDB(apiKey, title string) (*TMDBSearchResult, error) {
	params := url.Values{}
	params.Set("api_key", apiKey)
	params.Set("query", title)
	params.Set("language", "en-US")
	params.Set("page", "1")

	resp, err := http.Get(
		fmt.Sprintf("https://api.themoviedb.org/3/search/movie?%s", params.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("TMDB request failed: %w", err)
	}
	defer resp.Body.Close()

	var result TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode TMDB response: %w", err)
	}
	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no results found for: %s", title)
	}

	return &result.Results[0], nil
}

func extractYear(releaseDate string) string {
	if len(releaseDate) >= 4 {
		return releaseDate[:4]
	}
	return "Unknown"
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func sanitizeFilename(name string) string {
	r := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-",
		"*", "", "?", "", "\"", "",
		"<", "", ">", "", "|", "",
	)
	return strings.TrimSpace(r.Replace(name))
}

func prompt(question string) string {
	fmt.Print(question)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func findMKVFiles(dir string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) == ".mkv" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

// ── Pipeline Steps ────────────────────────────────────────────────────────────

type PipelineStep struct {
	Name string
	Done bool
}

type Pipeline struct {
	Steps   []PipelineStep
	Current int
}

func newPipeline(steps []string) *Pipeline {
	p := &Pipeline{}
	for _, s := range steps {
		p.Steps = append(p.Steps, PipelineStep{Name: s})
	}
	return p
}

func (p *Pipeline) printStatus() {
	fmt.Println()
	for i, step := range p.Steps {
		switch {
		case step.Done:
			fmt.Printf("  ✅ %s\n", step.Name)
		case i == p.Current:
			fmt.Printf("  ⏳ %s\n", step.Name)
		default:
			fmt.Printf("  ⬜ %s\n", step.Name)
		}
	}
	fmt.Println()
}

func (p *Pipeline) advance() {
	if p.Current < len(p.Steps) {
		p.Steps[p.Current].Done = true
		p.Current++
	}
}

// ── Ripping ───────────────────────────────────────────────────────────────────

func ripDisc(device, outputDir string, bar *progressbar.ProgressBar) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	source := fmt.Sprintf("dev:%s", device)
	cmd := exec.Command("makemkvcon", "mkv", source, "all", outputDir)

	// Pipe stderr so we can parse MakeMKV progress output
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stderr: %w", err)
	}
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("makemkvcon failed to start: %w", err)
	}

	// MakeMKV outputs lines like: PRGV:current,total,max
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRGV:") {
			var current, total, max int
			fmt.Sscanf(strings.TrimPrefix(line, "PRGV:"), "%d,%d,%d", &current, &total, &max)
			if max > 0 {
				_ = bar.Set(int(float64(current) / float64(max) * 100))
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("makemkvcon failed: %w", err)
	}

	_ = bar.Set(100)
	return nil
}

// ── File Copy with Progress ───────────────────────────────────────────────────

func copyFileWithProgress(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	bar := progressbar.NewOptions64(
		info.Size(),
		progressbar.OptionSetDescription("  Copying file "),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	_, err = io.Copy(io.MultiWriter(dstFile, bar), srcFile)
	return err
}

// ── SFTP Upload ───────────────────────────────────────────────────────────────

func uploadWithProgress(cfg Config, localFile, remotePath string) error {
	info, err := os.Stat(localFile)
	if err != nil {
		return fmt.Errorf("cannot stat local file: %w", err)
	}

	// Overall upload progress bar (byte-based)
	bar := progressbar.NewOptions64(
		info.Size(),
		progressbar.OptionSetDescription("  Uploading    "),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	// Create remote directory
	mkdirCmd := exec.Command(
		"ssh",
		"-p", cfg.SFTP.Port,
		"-i", cfg.SFTP.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", cfg.SFTP.User, cfg.SFTP.Host),
		fmt.Sprintf("mkdir -p '%s'", filepath.Dir(remotePath)),
	)
	mkdirCmd.Stderr = os.Stderr
	if err := mkdirCmd.Run(); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Use sftp in batch mode, piping the file ourselves so we can track progress
	remoteTarget := fmt.Sprintf(
		"%s@%s:%s",
		cfg.SFTP.User,
		cfg.SFTP.Host,
		remotePath,
	)
	scpCmd := exec.Command(
		"scp",
		"-P", cfg.SFTP.Port,
		"-i", cfg.SFTP.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		localFile,
		remoteTarget,
	)

	// We wrap the file read in a progress reader by replacing stdin isn't
	// viable with scp, so we poll file size on remote side as a proxy.
	// For true byte-level progress, we'd need a pure-Go SSH lib.
	// Instead: tick the bar based on elapsed time as an approximation,
	// then snap to 100% on completion.
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		start := time.Now()
		// Rough estimate: assume 50 MB/s average LAN speed
		const assumedBytesPerSec = 50 * 1024 * 1024
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				elapsed := time.Since(start).Seconds()
				estimated := int64(elapsed * assumedBytesPerSec)
				if estimated > info.Size() {
					estimated = info.Size() - 1
				}
				_ = bar.Set64(estimated)
			}
		}
	}()

	scpCmd.Stderr = os.Stderr
	err = scpCmd.Run()
	close(done)
	_ = bar.Set64(info.Size())
	fmt.Println()

	if err != nil {
		return fmt.Errorf("scp upload failed: %w", err)
	}

	return nil
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	// Load config
	cfgPath := "config.toml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}

	if cfg.TMDB.APIKey == "" {
		fmt.Println("❌ TMDB API key not set in config or TMDB_API_KEY env var.")
		os.Exit(1)
	}

	pipeline := newPipeline([]string{
		"Search TMDB",
		"Rip Disc",
		"Select Main Feature",
		"Rename File",
		"Upload to Server",
		"Cleanup",
	})

	fmt.Println("=== 🎬 DVD/Blu-ray Ripper for Jellyfin ===")
	pipeline.printStatus()

	// ── Step 1: TMDB Search ──────────────────────────────────────────────────
	rawTitle := prompt("Enter movie title: ")
	if rawTitle == "" {
		fmt.Println("❌ No title entered.")
		os.Exit(1)
	}

	fmt.Printf("\n🔍 Searching TMDB for \"%s\"...\n", rawTitle)
	result, err := searchTMDB(cfg.TMDB.APIKey, rawTitle)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}

	year := extractYear(result.ReleaseDate)
	movieName := sanitizeFilename(result.Title)
	folderName := fmt.Sprintf("%s (%s)", movieName, year)

	fmt.Printf("✅ Found: %s (%s)\n", result.Title, year)
	confirm := prompt(fmt.Sprintf("Use \"%s\"? (y/n): ", folderName))
	if strings.ToLower(confirm) != "y" {
		custom := prompt("Custom name (without year): ")
		customYear := prompt("Year: ")
		movieName = sanitizeFilename(custom)
		year = customYear
		folderName = fmt.Sprintf("%s (%s)", movieName, year)
	}

	pipeline.advance()
	pipeline.printStatus()

	// ── Step 2: Rip Disc ─────────────────────────────────────────────────────
	ripOutputDir := filepath.Join(cfg.Output.Dir, folderName)
	fmt.Printf("💿 Ripping disc from %s...\n\n", cfg.Drive.Device)

	ripBar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("  Ripping      "),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	if err := ripDisc(cfg.Drive.Device, ripOutputDir, ripBar); err != nil {
		fmt.Printf("\n❌ Rip failed: %v\n", err)
		os.Exit(1)
	}

	pipeline.advance()
	pipeline.printStatus()

	// ── Step 3: Select Main Feature ──────────────────────────────────────────
	mkvFiles, err := findMKVFiles(ripOutputDir)
	if err != nil || len(mkvFiles) == 0 {
		fmt.Println("❌ No MKV files found after ripping.")
		os.Exit(1)
	}

	var mainMKV string
	if len(mkvFiles) == 1 {
		mainMKV = mkvFiles[0]
		fmt.Printf("📼 Found 1 MKV: %s\n", filepath.Base(mainMKV))
	} else {
		fmt.Println("\n📼 Multiple MKV files found:")
		for i, f := range mkvFiles {
			info, _ := os.Stat(f)
			sizeMB := int64(0)
			if info != nil {
				sizeMB = info.Size() / 1024 / 1024
			}
			fmt.Printf("  [%d] %s (%d MB)\n", i+1, filepath.Base(f), sizeMB)
		}
		choice := prompt("Select the main feature [number]: ")
		idx := 0
		fmt.Sscanf(choice, "%d", &idx)
		if idx < 1 || idx > len(mkvFiles) {
			fmt.Println("❌ Invalid selection.")
			os.Exit(1)
		}
		mainMKV = mkvFiles[idx-1]
	}

	pipeline.advance()
	pipeline.printStatus()

	// ── Step 4: Rename File ──────────────────────────────────────────────────
	finalFilename := fmt.Sprintf("%s.mkv", folderName)
	finalLocalPath := filepath.Join(ripOutputDir, finalFilename)

	if mainMKV != finalLocalPath {
		fmt.Printf("✏️  Renaming to: %s\n", finalFilename)
		if err := copyFileWithProgress(mainMKV, finalLocalPath); err != nil {
			fmt.Printf("❌ Failed to copy file: %v\n", err)
			os.Exit(1)
		}
		os.Remove(mainMKV)
	}

	pipeline.advance()
	pipeline.printStatus()

	// ── Step 5: Upload ───────────────────────────────────────────────────────
	if cfg.SFTP.Host != "" && cfg.SFTP.User != "" {
		remoteFolderPath := filepath.ToSlash(
			filepath.Join(cfg.SFTP.RemotePath, folderName),
		)
		remoteFilePath := fmt.Sprintf("%s/%s", remoteFolderPath, finalFilename)

		fmt.Printf("📤 Uploading to %s...\n\n", cfg.SFTP.Host)
		if err := uploadWithProgress(cfg, finalLocalPath, remoteFilePath); err != nil {
			fmt.Printf("❌ Upload failed: %v\n", err)
			fmt.Printf("📁 File saved locally: %s\n", finalLocalPath)
			os.Exit(1)
		}

		pipeline.advance()
		pipeline.printStatus()

		// ── Step 6: Cleanup ──────────────────────────────────────────────────
		cleanupChoice := prompt("🧹 Delete local temp files? (y/n): ")
		if strings.ToLower(cleanupChoice) == "y" {
			if err := os.RemoveAll(ripOutputDir); err != nil {
				fmt.Printf("⚠️  Cleanup failed: %v\n", err)
			} else {
				fmt.Println("✅ Local files removed.")
			}
		}
	} else {
		fmt.Printf("\n📁 SFTP not configured. File saved at:\n   %s\n", finalLocalPath)
		pipeline.advance()
	}

	pipeline.advance()
	pipeline.printStatus()

	fmt.Printf("🎉 Done! \"%s\" is ready in Jellyfin.\n\n", folderName)
}
