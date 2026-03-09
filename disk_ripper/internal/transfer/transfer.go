package transfer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"ripper/internal/config"
)

// CopyFile copies src to dst, streaming bytes written to progressCh.
// progressCh may be nil.
func CopyFile(src, dst string, progressCh chan<- int64) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if progressCh == nil {
		_, err = io.Copy(dstFile, srcFile)
		return err
	}

	buf := make([]byte, 32*1024)
	var written int64
	for {
		nr, readErr := srcFile.Read(buf)
		if nr > 0 {
			nw, writeErr := dstFile.Write(buf[:nr])
			written += int64(nw)
			select {
			case progressCh <- written:
			default:
			}
			if writeErr != nil {
				return writeErr
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

// Upload copies localFile to the remote server via scp, streaming estimated
// bytes transferred to progressCh. progressCh may be nil.
func Upload(cfg config.SFTPConfig, localFile, remotePath string, progressCh chan<- int64) error {
	info, err := os.Stat(localFile)
	if err != nil {
		return fmt.Errorf("cannot stat local file: %w", err)
	}

	// Create remote directory
	mkdirCmd := exec.Command(
		"ssh",
		"-p", cfg.Port,
		"-i", cfg.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", cfg.User, cfg.Host),
		fmt.Sprintf("mkdir -p '%s'", filepath.Dir(remotePath)),
	)
	mkdirCmd.Stderr = os.Stderr
	if err := mkdirCmd.Run(); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	remoteTarget := fmt.Sprintf("%s@%s:%s", cfg.User, cfg.Host, remotePath)
	scpCmd := exec.Command(
		"scp",
		"-P", cfg.Port,
		"-i", cfg.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		localFile,
		remoteTarget,
	)
	scpCmd.Stderr = os.Stderr

	if progressCh != nil {
		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			start := time.Now()
			const assumedBytesPerSec = 50 * 1024 * 1024
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					elapsed := time.Since(start).Seconds()
					estimated := int64(elapsed * assumedBytesPerSec)
					if estimated > info.Size()-1 {
						estimated = info.Size() - 1
					}
					select {
					case progressCh <- estimated:
					default:
					}
				}
			}
		}()
		err = scpCmd.Run()
		close(done)
		select {
		case progressCh <- info.Size():
		default:
		}
	} else {
		err = scpCmd.Run()
	}

	if err != nil {
		return fmt.Errorf("scp upload failed: %w", err)
	}
	return nil
}
