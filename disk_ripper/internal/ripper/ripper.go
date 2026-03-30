package ripper

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	types "ripper/internal"
)

// RipDisc runs makemkvcon and streams progress (0–100) to progressCh.
// The caller is responsible for closing or draining progressCh after the
// returned error is received.
func RipDisc(device, title string, outputDir string, progressCh chan<- int) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	source := fmt.Sprintf("dev:%s", device)
	cmd := exec.Command("makemkvcon", "--robot", "--progress=-stderr", "mkv", source, title, outputDir)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stderr: %w", err)
	}
	cmd.Stdout = io.Discard

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
			if max > 0 && progressCh != nil {
				pct := int(float64(current) / float64(max) * 100)
				select {
				case progressCh <- pct:
				default:
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("makemkvcon failed: %w", err)
	}

	if progressCh != nil {
		select {
		case progressCh <- 100:
		default:
		}
	}
	return nil
}

// FindMKVFiles returns all .mkv files in dir (non-recursive).
func FindMKVFiles(dir string) ([]string, error) {
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

// FindTitles runs makemkvcon to list available titles on a disc.
func FindTitles(device string, progressCh chan<- types.FetchingProgress) error {
	titleStore := make(map[int]types.TitleInfo)

	source := fmt.Sprintf("dev:%s", device)
	cmd := exec.Command(
		"makemkvcon",
		"--robot",
		"--messages=-stderr",
		"--progress=-stderr",
		"info",
		source,
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stderr: %w", err)
	}
	cmd.Stdout = io.Discard

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("makemkvcon failed to start: %w", err)
	}

	// MakeMKV outputs lines like: PRGV:current,total,max
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "TINFO:") {
			entry, ok := parseTINFO(line)
			if ok {
				updateTitleStore(titleStore, entry)
			}
		}

		if strings.HasPrefix(line, "PRGV:") {
			var current, total, max int
			_, _ = fmt.Sscanf(
				strings.TrimPrefix(line, "PRGV:"),
				"%d,%d,%d",
				&current,
				&total,
				&max,
			)

			if max > 0 && progressCh != nil {
				pct := int(float64(current) / float64(max) * 100)
				tempProgress := types.FetchingProgress{
					Pct:    pct,
					Titles: titleStore,
				}
				select {
				case progressCh <- tempProgress:
				default:
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("makemkvcon failed: %w", err)
	}

	if progressCh != nil {
		tempProgress := types.FetchingProgress{
			Pct:    100,
			Titles: titleStore,
		}
		select {
		case progressCh <- tempProgress:
		default:
		}
	}

	return nil
}

func SliceTitleStore(titleStore map[int]types.TitleInfo) []types.TitleInfo {
	titles := make([]types.TitleInfo, 0, len(titleStore))
	for _, t := range titleStore {
		titles = append(titles, t)
	}

	sort.Slice(titles, func(i, j int) bool {
		return titles[i].ID < titles[j].ID
	})
	return titles
}

func parseTINFO(line string) (types.TitleEntry, bool) {
	// Example:
	// TINFO:23,2,0,"Fantastic Beasts and Where to Find Them"

	payload := strings.TrimPrefix(line, "TINFO:")
	parts := strings.SplitN(payload, ",", 4)
	if len(parts) != 4 {
		return types.TitleEntry{}, false
	}

	trackID, err := strconv.Atoi(parts[0])
	if err != nil {
		return types.TitleEntry{}, false
	}

	detailInt, err := strconv.Atoi(parts[1])
	if err != nil {
		return types.TitleEntry{}, false
	}

	trackDetail := types.TitleDetail(detailInt)

	value, err := strconv.Unquote(parts[3])
	if err != nil {
		return types.TitleEntry{}, false
	}

	return types.TitleEntry{
		TitleID:     trackID,
		TitleDetail: trackDetail,
		TitleValue:  value,
	}, true
}

func updateTitleStore(store map[int]types.TitleInfo, entry types.TitleEntry) {
	t := store[entry.TitleID]
	t.ID = entry.TitleID

	applyTitleEntry(&t, entry)

	store[entry.TitleID] = t
}

func applyTitleEntry(t *types.TitleInfo, entry types.TitleEntry) {
	switch entry.TitleDetail {
	case types.TD_ChapterCount:
		if n, err := strconv.Atoi(entry.TitleValue); err == nil {
			t.Chapters = n
		}
	case types.TD_FileSizeBytes:
		if n, err := strconv.ParseInt(entry.TitleValue, 10, 64); err == nil {
			t.SizeBytes = n
		}
	case types.TD_FileSizeGB:
		t.SizeHuman = entry.TitleValue
	case types.TD_DiskTitle:
		t.Name = entry.TitleValue
	case types.TD_LengthInSeconds:
		t.Duration = entry.TitleValue
	}
}
