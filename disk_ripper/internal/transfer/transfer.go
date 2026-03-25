package transfer

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"ripper/internal/config"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
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
	socket := os.Getenv("SSH_AUTH_SOCK")
	agentConn, err := net.Dial("unix", socket)
	if err != nil {
		return fmt.Errorf("connect to ssh agent: %w", err)
	}
	defer agentConn.Close()

	hostKeyCallback, err := knownhosts.New(os.ExpandEnv("$HOME/.ssh/known_hosts"))
	if err != nil {
		return fmt.Errorf("load known_hosts: %w", err)
	}

	sshCfg := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	sshConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", cfg.Host, cfg.Port), sshCfg)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer sshConn.Close()

	client, err := sftp.NewClient(sshConn)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer client.Close()

	if err := client.MkdirAll(filepath.Dir(remotePath)); err != nil {
		return fmt.Errorf("mkdir remote: %w", err)
	}

	src, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer src.Close()

	dst, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file: %w", err)
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
