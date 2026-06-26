package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/littepointR/mocktrue/internal/core/platform"
)

func TestServiceSaveReadAndRememberLastWorkspace(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(&platform.Paths{ConfigDir: dir, DataDir: dir})
	path := filepath.Join(dir, "session.portweave.json")
	content := `{"kind":"mocktrue.workspace.v1","settings":{"global":{"Theme":"dark"}}}`

	if err := svc.SaveWorkspace(context.Background(), path, content); err != nil {
		t.Fatalf("SaveWorkspace: %v", err)
	}

	loaded, err := svc.ReadWorkspace(context.Background(), path)
	if err != nil {
		t.Fatalf("ReadWorkspace: %v", err)
	}
	if loaded.Path != path || loaded.Content != content {
		t.Fatalf("ReadWorkspace = %+v, want path %q content %q", loaded, path, content)
	}

	last, err := svc.LoadLastWorkspace(context.Background())
	if err != nil {
		t.Fatalf("LoadLastWorkspace: %v", err)
	}
	if !last.Found || last.Path != path || last.Content != content {
		t.Fatalf("LoadLastWorkspace = %+v, want found path/content", last)
	}
}

func TestServiceLoadLastWorkspaceMissingFileReturnsCleanEmptyResult(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(&platform.Paths{ConfigDir: dir, DataDir: dir})
	missing := filepath.Join(dir, "missing.portweave.json")

	if err := svc.RememberLastWorkspace(context.Background(), missing); err != nil {
		t.Fatalf("RememberLastWorkspace: %v", err)
	}

	last, err := svc.LoadLastWorkspace(context.Background())
	if err != nil {
		t.Fatalf("LoadLastWorkspace should not fail for missing file: %v", err)
	}
	if last.Found || last.Path != missing || last.Content != "" {
		t.Fatalf("LoadLastWorkspace missing = %+v, want not found with remembered path", last)
	}
}

func TestServiceExportWorkspaceDoesNotRememberLastWorkspace(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(&platform.Paths{ConfigDir: dir, DataDir: dir})
	activePath := filepath.Join(dir, "active.portweave.json")
	copyPath := filepath.Join(dir, "copy.portweave.json")

	if err := svc.SaveWorkspace(context.Background(), activePath, `{"active":true}`); err != nil {
		t.Fatalf("SaveWorkspace: %v", err)
	}
	if err := svc.ExportWorkspace(context.Background(), copyPath, `{"copy":true}`); err != nil {
		t.Fatalf("ExportWorkspace: %v", err)
	}

	copyFile, err := svc.ReadWorkspace(context.Background(), copyPath)
	if err != nil {
		t.Fatalf("ReadWorkspace copy: %v", err)
	}
	if copyFile.Content != `{"copy":true}` {
		t.Fatalf("ExportWorkspace content = %q, want copy content", copyFile.Content)
	}
	last, err := svc.LoadLastWorkspace(context.Background())
	if err != nil {
		t.Fatalf("LoadLastWorkspace: %v", err)
	}
	if !last.Found || last.Path != activePath || last.Content != `{"active":true}` {
		t.Fatalf("LoadLastWorkspace after export = %+v, want active workspace unchanged", last)
	}
}

func TestServiceSelectWorkspaceOpenPathUsesCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(&platform.Paths{ConfigDir: dir})
	currentPath := filepath.Join(dir, "nested", "session.portweave.json")
	selectedPath := filepath.Join(dir, "selected.portweave.json")
	var gotDir string
	svc.selectOpenPathFunc = func(directory string) (string, error) {
		gotDir = directory
		return selectedPath, nil
	}

	path, err := svc.SelectWorkspaceOpenPath(context.Background(), currentPath)
	if err != nil {
		t.Fatalf("SelectWorkspaceOpenPath: %v", err)
	}
	if path != selectedPath {
		t.Fatalf("SelectWorkspaceOpenPath = %q, want %q", path, selectedPath)
	}
	if gotDir != filepath.Dir(currentPath) {
		t.Fatalf("dialog directory = %q, want %q", gotDir, filepath.Dir(currentPath))
	}
}

func TestServiceSelectWorkspaceSavePathUsesDefaultWhenNoCurrentPath(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(&platform.Paths{ConfigDir: dir})
	selectedPath := filepath.Join(dir, "selected.portweave.json")
	var gotDir string
	var gotFilename string
	svc.selectSavePathFunc = func(directory string, filename string) (string, error) {
		gotDir = directory
		gotFilename = filename
		return selectedPath, nil
	}

	path, err := svc.SelectWorkspaceSavePath(context.Background(), "")
	if err != nil {
		t.Fatalf("SelectWorkspaceSavePath: %v", err)
	}
	if path != selectedPath {
		t.Fatalf("SelectWorkspaceSavePath = %q, want %q", path, selectedPath)
	}
	if gotDir != dir {
		t.Fatalf("dialog directory = %q, want %q", gotDir, dir)
	}
	if gotFilename != "workspace.portweave.json" {
		t.Fatalf("dialog filename = %q, want workspace.portweave.json", gotFilename)
	}
}

func TestServiceRejectsEmptyWorkspacePath(t *testing.T) {
	svc := NewService(&platform.Paths{ConfigDir: t.TempDir()})

	if err := svc.SaveWorkspace(context.Background(), "", "{}"); err == nil {
		t.Fatalf("SaveWorkspace must reject empty path")
	}
	if err := svc.ExportWorkspace(context.Background(), "", "{}"); err == nil {
		t.Fatalf("ExportWorkspace must reject empty path")
	}
	if _, err := svc.ReadWorkspace(context.Background(), ""); err == nil {
		t.Fatalf("ReadWorkspace must reject empty path")
	}
	if err := svc.RememberLastWorkspace(context.Background(), ""); err == nil {
		t.Fatalf("RememberLastWorkspace must reject empty path")
	}
}

func TestServiceNameAndDefaultWorkspacePathUseConfigFallback(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)
	if got := svc.ServiceName(); got != "workspace" {
		t.Fatalf("ServiceName = %q, want workspace", got)
	}

	path, err := svc.DefaultWorkspacePath(context.Background())
	if err != nil {
		t.Fatalf("DefaultWorkspacePath: %v", err)
	}
	if path != "workspace.portweave.json" {
		t.Fatalf("DefaultWorkspacePath with nil paths = %q, want workspace.portweave.json", path)
	}
}

func TestWorkspaceDialogDirectoryFallsBackWhenPathIsEmpty(t *testing.T) {
	t.Parallel()
	if got := workspaceDialogDirectory("", "/tmp/fallback"); got != "/tmp/fallback" {
		t.Fatalf("workspaceDialogDirectory empty path = %q, want fallback", got)
	}
	if got := workspaceDialogDirectory("", ""); got != "." {
		t.Fatalf("workspaceDialogDirectory empty path and fallback = %q, want .", got)
	}
}

func TestSelectWorkspaceDialogHelpersRequireApplication(t *testing.T) {
	t.Parallel()
	if _, err := selectWorkspaceOpenPath(t.TempDir()); err == nil {
		t.Fatalf("selectWorkspaceOpenPath without Wails app must error")
	}
	if _, err := selectWorkspaceSavePath(t.TempDir(), "workspace.portweave.json"); err == nil {
		t.Fatalf("selectWorkspaceSavePath without Wails app must error")
	}
}

func TestServiceLoadLastWorkspaceInvalidStateReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := NewService(&platform.Paths{ConfigDir: dir})
	if err := os.WriteFile(filepath.Join(dir, stateFileName), []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid state fixture: %v", err)
	}

	if _, err := svc.LoadLastWorkspace(context.Background()); err == nil {
		t.Fatalf("LoadLastWorkspace with invalid state JSON must error")
	}
}

func TestServiceMethodsReturnContextErrorBeforeIO(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := NewService(&platform.Paths{ConfigDir: dir})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	path := filepath.Join(dir, "workspace.portweave.json")

	if err := svc.SaveWorkspace(ctx, path, "{}"); err == nil {
		t.Fatalf("SaveWorkspace with canceled context must error")
	}
	if err := svc.ExportWorkspace(ctx, path, "{}"); err == nil {
		t.Fatalf("ExportWorkspace with canceled context must error")
	}
	if _, err := svc.ReadWorkspace(ctx, path); err == nil {
		t.Fatalf("ReadWorkspace with canceled context must error")
	}
	if err := svc.RememberLastWorkspace(ctx, path); err == nil {
		t.Fatalf("RememberLastWorkspace with canceled context must error")
	}
	if _, err := svc.LoadLastWorkspace(ctx); err == nil {
		t.Fatalf("LoadLastWorkspace with canceled context must error")
	}
	if _, err := svc.DefaultWorkspacePath(ctx); err == nil {
		t.Fatalf("DefaultWorkspacePath with canceled context must error")
	}
	if _, err := svc.SelectWorkspaceOpenPath(ctx, path); err == nil {
		t.Fatalf("SelectWorkspaceOpenPath with canceled context must error")
	}
	if _, err := svc.SelectWorkspaceSavePath(ctx, path); err == nil {
		t.Fatalf("SelectWorkspaceSavePath with canceled context must error")
	}
}
