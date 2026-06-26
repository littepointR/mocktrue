package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	mterrors "github.com/littepointR/portweave/internal/core/errors"
	"github.com/littepointR/portweave/internal/core/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const stateFileName = "workspace-state.json"

// FileResult is a workspace JSON file read result.
type FileResult struct {
	Path    string
	Content string
}

// LastWorkspaceResult is returned during startup auto-load.
type LastWorkspaceResult struct {
	Found   bool
	Path    string
	Content string
}

// Service exposes workspace JSON file persistence to the frontend.
type Service struct {
	paths              *platform.Paths
	selectOpenPathFunc func(directory string) (string, error)
	selectSavePathFunc func(directory string, filename string) (string, error)
}

// NewService constructs a workspace file service.
func NewService(paths *platform.Paths) *Service {
	return &Service{
		paths:              paths,
		selectOpenPathFunc: selectWorkspaceOpenPath,
		selectSavePathFunc: selectWorkspaceSavePath,
	}
}

func (s *Service) ServiceName() string { return "workspace" }

// SaveWorkspace writes content to path and remembers it as the last workspace.
func (s *Service) SaveWorkspace(ctx context.Context, path string, content string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if path == "" {
		return mterrors.New(mterrors.CodeInvalid, "workspace path must not be empty")
	}
	if err := writeAtomic(path, []byte(content), 0o600); err != nil {
		return err
	}
	return s.RememberLastWorkspace(ctx, path)
}

// ExportWorkspace writes content to path without changing the last workspace.
func (s *Service) ExportWorkspace(ctx context.Context, path string, content string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if path == "" {
		return mterrors.New(mterrors.CodeInvalid, "workspace path must not be empty")
	}
	return writeAtomic(path, []byte(content), 0o600)
}

// ReadWorkspace reads a workspace JSON file from path.
func (s *Service) ReadWorkspace(ctx context.Context, path string) (*FileResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, mterrors.New(mterrors.CodeInvalid, "workspace path must not be empty")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, mterrors.Wrap(mterrors.CodeIO, "read workspace file", err)
	}
	return &FileResult{Path: path, Content: string(content)}, nil
}

// SelectWorkspaceOpenPath prompts for a workspace JSON file to open.
func (s *Service) SelectWorkspaceOpenPath(ctx context.Context, currentPath string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return s.selectOpenPathFunc(workspaceDialogDirectory(currentPath, s.configDir()))
}

// SelectWorkspaceSavePath prompts for a workspace JSON file save target.
func (s *Service) SelectWorkspaceSavePath(ctx context.Context, currentPath string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	defaultPath, err := s.DefaultWorkspacePath(ctx)
	if err != nil {
		return "", err
	}
	basePath := currentPath
	if basePath == "" {
		basePath = defaultPath
	}
	return s.selectSavePathFunc(workspaceDialogDirectory(basePath, s.configDir()), filepath.Base(basePath))
}

// RememberLastWorkspace stores the last workspace file path for startup load.
func (s *Service) RememberLastWorkspace(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if path == "" {
		return mterrors.New(mterrors.CodeInvalid, "workspace path must not be empty")
	}
	state := lastWorkspaceState{Path: path}
	data, err := json.Marshal(state)
	if err != nil {
		return mterrors.Wrap(mterrors.CodeInternal, "marshal workspace state", err)
	}
	return writeAtomic(s.statePath(), data, 0o600)
}

// LoadLastWorkspace loads the remembered workspace file. Missing files are not
// fatal: the app should start clean if the previous file is unavailable.
func (s *Service) LoadLastWorkspace(ctx context.Context) (*LastWorkspaceResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	state, err := s.loadState()
	if err != nil {
		return nil, err
	}
	if state.Path == "" {
		return &LastWorkspaceResult{}, nil
	}
	content, err := os.ReadFile(state.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LastWorkspaceResult{Path: state.Path}, nil
		}
		return nil, mterrors.Wrap(mterrors.CodeIO, "read last workspace file", err)
	}
	return &LastWorkspaceResult{Found: true, Path: state.Path, Content: string(content)}, nil
}

// DefaultWorkspacePath returns the built-in workspace save location.
func (s *Service) DefaultWorkspacePath(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return filepath.Join(s.configDir(), "workspace.portweave.json"), nil
}

type lastWorkspaceState struct {
	Path string `json:"path"`
}

func (s *Service) loadState() (lastWorkspaceState, error) {
	data, err := os.ReadFile(s.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return lastWorkspaceState{}, nil
		}
		return lastWorkspaceState{}, mterrors.Wrap(mterrors.CodeIO, "read workspace state", err)
	}
	var state lastWorkspaceState
	if err := json.Unmarshal(data, &state); err != nil {
		return lastWorkspaceState{}, mterrors.Wrap(mterrors.CodeInvalid, "parse workspace state", err)
	}
	return state, nil
}

func (s *Service) statePath() string {
	return filepath.Join(s.configDir(), stateFileName)
}

func (s *Service) configDir() string {
	if s.paths != nil && s.paths.ConfigDir != "" {
		return s.paths.ConfigDir
	}
	return "."
}

func workspaceDialogDirectory(path string, fallback string) string {
	if path != "" {
		return filepath.Dir(path)
	}
	if fallback != "" {
		return fallback
	}
	return "."
}

func selectWorkspaceOpenPath(directory string) (string, error) {
	app := application.Get()
	if app == nil {
		return "", mterrors.New(mterrors.CodePlatform, "application is not available")
	}
	return app.Dialog.OpenFile().
		SetTitle("选择配置文件").
		SetMessage("选择 PortWeave 配置文件").
		SetDirectory(directory).
		SetButtonText("选择").
		AddFilter("JSON 配置文件", "*.json").
		PromptForSingleSelection()
}

func selectWorkspaceSavePath(directory string, filename string) (string, error) {
	app := application.Get()
	if app == nil {
		return "", mterrors.New(mterrors.CodePlatform, "application is not available")
	}
	return app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:                "保存配置文件",
		Message:              "保存 PortWeave 配置文件",
		Directory:            directory,
		Filename:             filename,
		ButtonText:           "保存",
		CanCreateDirectories: true,
		Filters: []application.FileFilter{
			{DisplayName: "JSON 配置文件", Pattern: "*.json"},
		},
	}).PromptForSingleSelection()
}

func writeAtomic(path string, data []byte, perm os.FileMode) error {
	if path == "" {
		return mterrors.New(mterrors.CodeInvalid, "path must not be empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return mterrors.Wrap(mterrors.CodePlatform, "create workspace directory", err)
	}
	tmp, err := os.CreateTemp(dir, ".workspace-*.tmp")
	if err != nil {
		return mterrors.Wrap(mterrors.CodeIO, "create temp workspace", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if _, statErr := os.Stat(tmpName); statErr == nil {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return mterrors.Wrap(mterrors.CodeIO, "write temp workspace", err)
	}
	if err := tmp.Close(); err != nil {
		return mterrors.Wrap(mterrors.CodeIO, "close temp workspace", err)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return mterrors.Wrap(mterrors.CodeIO, "chmod temp workspace", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return mterrors.Wrap(mterrors.CodeIO, fmt.Sprintf("rename temp workspace to %s", path), err)
	}
	return nil
}
