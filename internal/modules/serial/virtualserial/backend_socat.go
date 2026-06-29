//go:build linux || darwin || freebsd || netbsd || openbsd

package virtualserial

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/littepointR/portweave/internal/core/errors"
)

type socatBackend struct{}

// DefaultBackend returns the platform virtual serial backend.
func DefaultBackend() Backend {
	return socatBackend{}
}

func (socatBackend) Name() string { return "socat" }

func (socatBackend) Status(context.Context) BackendStatus {
	if _, err := exec.LookPath("socat"); err != nil {
		return BackendStatus{
			Name:      "socat",
			Available: false,
			Message:   "socat is not installed or is not on PATH",
			Reason:    err.Error(),
		}
	}
	return BackendStatus{
		Name:      "socat",
		Available: true,
		Message:   "socat virtual serial backend is available",
	}
}

func (socatBackend) CreatePair(ctx context.Context, pairID, port1Name, port2Name string) (*VirtualPair, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, errors.Wrap(errors.CodeIO, "create virtual serial pair", err)
		}
	}

	port1Path, port2Path, pairDir, err := socatEndpointPaths(port1Name, port2Name)
	if err != nil {
		return nil, err
	}
	cleanup := func() {
		_ = os.Remove(port1Path)
		_ = os.Remove(port2Path)
		_ = os.Remove(pairDir)
	}

	cmd := exec.Command("socat", "-d", "-d",
		fmt.Sprintf("pty,raw,echo=0,link=%s", port1Path),
		fmt.Sprintf("pty,raw,echo=0,link=%s", port2Path))
	if err := cmd.Start(); err != nil {
		cleanup()
		return nil, errors.Wrap(errors.CodeIO, "start socat", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fileExists(port1Path) && fileExists(port2Path) {
			return newVirtualPair(pairID, port1Path, port2Path, socatPairHandle{
				cmd:   cmd,
				paths: []string{port1Path, port2Path},
				dir:   pairDir,
			}), nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	cleanup()
	return nil, errors.New(errors.CodeIO, "timeout waiting for socat symlinks")
}

func (b socatBackend) CreatePort(ctx context.Context, portID, publicName string) (*VirtualPair, error) {
	return b.CreatePair(ctx, portID, publicName, publicName+"-peer")
}

func socatEndpointPaths(port1Name, port2Name string) (string, string, string, error) {
	port1Base, err := socatEndpointName(port1Name)
	if err != nil {
		return "", "", "", err
	}
	port2Base, err := socatEndpointName(port2Name)
	if err != nil {
		return "", "", "", err
	}
	if port1Base == port2Base {
		return "", "", "", errors.New(errors.CodeInvalid, "virtual serial endpoint names must be distinct")
	}

	pairDir, err := os.MkdirTemp("", "portweave-virtualserial-*")
	if err != nil {
		return "", "", "", errors.Wrap(errors.CodeIO, "create virtual serial temp directory", err)
	}
	return filepath.Join(pairDir, port1Base), filepath.Join(pairDir, port2Base), pairDir, nil
}

func socatEndpointName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errors.New(errors.CodeInvalid, "virtual serial endpoint name must not be empty")
	}
	if filepath.IsAbs(trimmed) {
		return "", errors.New(errors.CodeInvalid, "virtual serial endpoint name must be relative")
	}
	if strings.Contains(trimmed, "..") {
		return "", errors.New(errors.CodeInvalid, "virtual serial endpoint name must not contain '..'")
	}
	if strings.ContainsAny(trimmed, `/\`) {
		return "", errors.New(errors.CodeInvalid, "virtual serial endpoint name must not contain path separators")
	}
	if filepath.Clean(trimmed) != trimmed || trimmed == "." {
		return "", errors.New(errors.CodeInvalid, "virtual serial endpoint name must be a simple file name")
	}
	for _, r := range trimmed {
		allowed := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' || r == '.'
		if !allowed {
			return "", errors.New(errors.CodeInvalid, "virtual serial endpoint name contains unsupported characters")
		}
	}
	return trimmed, nil
}

type socatPairHandle struct {
	cmd   *exec.Cmd
	paths []string
	dir   string
}

func (h socatPairHandle) Stop() error {
	if h.cmd != nil && h.cmd.Process != nil {
		_ = h.cmd.Process.Kill()
		_ = h.cmd.Wait()
	}
	for _, path := range h.paths {
		_ = os.Remove(path)
	}
	if h.dir != "" {
		_ = os.Remove(h.dir)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
