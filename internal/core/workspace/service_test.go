package workspace

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/littepointR/mocktrue/internal/core/platform"
)

func TestServiceSaveReadAndRememberLastWorkspace(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(&platform.Paths{ConfigDir: dir, DataDir: dir})
	path := filepath.Join(dir, "session.mocktrue.json")
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
	missing := filepath.Join(dir, "missing.mocktrue.json")

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
	activePath := filepath.Join(dir, "active.mocktrue.json")
	copyPath := filepath.Join(dir, "copy.mocktrue.json")

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
	currentPath := filepath.Join(dir, "nested", "session.mocktrue.json")
	selectedPath := filepath.Join(dir, "selected.mocktrue.json")
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
	selectedPath := filepath.Join(dir, "selected.mocktrue.json")
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
	if gotFilename != "workspace.mocktrue.json" {
		t.Fatalf("dialog filename = %q, want workspace.mocktrue.json", gotFilename)
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
