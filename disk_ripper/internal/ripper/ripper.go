package ripper

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RipDisc runs makemkvcon and streams progress (0–100) to progressCh.
// The caller is responsible for closing or draining progressCh after the
// returned error is received.
func RipDisc(device, outputDir string, progressCh chan<- int) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	source := fmt.Sprintf("dev:%s", device)
	cmd := exec.Command("makemkvcon", "--robot", "mkv", source, "all", outputDir)

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
