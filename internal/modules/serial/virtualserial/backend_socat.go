//go:build linux || darwin || freebsd || netbsd || openbsd

package virtualserial

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	port1Path := socatEndpointPath(port1Name)
	port2Path := socatEndpointPath(port2Name)

	_ = os.Remove(port1Path)
	_ = os.Remove(port2Path)

	cmd := exec.Command("socat", "-d", "-d",
		fmt.Sprintf("pty,raw,echo=0,link=%s", port1Path),
		fmt.Sprintf("pty,raw,echo=0,link=%s", port2Path))
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(errors.CodeIO, "start socat", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fileExists(port1Path) && fileExists(port2Path) {
			return newVirtualPair(pairID, port1Path, port2Path, socatPairHandle{
				cmd:   cmd,
				paths: []string{port1Path, port2Path},
			}), nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	_ = os.Remove(port1Path)
	_ = os.Remove(port2Path)
	return nil, errors.New(errors.CodeIO, "timeout waiting for socat symlinks")
}

func (b socatBackend) CreatePort(ctx context.Context, portID, publicName string) (*VirtualPair, error) {
	return b.CreatePair(ctx, portID, publicName, publicName+"-peer")
}

func socatEndpointPath(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(os.TempDir(), name)
}

type socatPairHandle struct {
	cmd   *exec.Cmd
	paths []string
}

func (h socatPairHandle) Stop() error {
	if h.cmd != nil && h.cmd.Process != nil {
		_ = h.cmd.Process.Kill()
		_ = h.cmd.Wait()
	}
	for _, path := range h.paths {
		_ = os.Remove(path)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
