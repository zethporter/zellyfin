package transfer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// Upload copies localFile to serverPath on the local filesystem, streaming
// estimated bytes transferred to progressCh. progressCh may be nil.
func Upload(localFile, serverPath string, progressCh chan<- int64) error {
	if err := os.MkdirAll(filepath.Dir(serverPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	src, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(serverPath)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer dst.Close()

	if progressCh == nil {
		_, err = io.Copy(dst, src)
		return err
	}

	buf := make([]byte, 32*1024)
	var written int64
	for {
		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := dst.Write(buf[:nr])
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
