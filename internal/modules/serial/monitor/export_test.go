package monitor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleFrames() []Frame {
	return []Frame{
		enrichFrame(Frame{
			Seq:       1,
			Timestamp: time.Date(2026, 6, 20, 1, 2, 3, 4*1e6, time.UTC),
			Direction: DirectionAToB,
			Port:      "/tmp/a",
			Data:      []byte("hello"),
		}, "utf-8"),
		enrichFrame(Frame{
			Seq:       2,
			Timestamp: time.Date(2026, 6, 20, 1, 2, 4, 4*1e6, time.UTC),
			Direction: DirectionBToA,
			Port:      "/tmp/b",
			Data:      []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0a, 0xc5, 0xcd},
		}, "utf-8"),
	}
}

func TestExportFramesCSVTextAndHTML(t *testing.T) {
	formats := map[string]string{
		ExportCSV:  ".csv",
		ExportText: ".txt",
		ExportHTML: ".html",
	}
	for format, ext := range formats {
		t.Run(format, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "capture"+ext)
			got, err := exportFrames(ExportRequest{MonitorID: "m1", Format: format, Path: path}, sampleFrames())
			if err != nil {
				t.Fatalf("exportFrames: %v", err)
			}
			if got != path {
				t.Fatalf("path = %q, want %q", got, path)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			text := string(content)
			if !strings.Contains(text, "a_to_b") || !strings.Contains(text, "hello") {
				t.Fatalf("export content missing frame data: %s", text)
			}
			if !strings.Contains(text, "读保持寄存器") {
				t.Fatalf("export content missing modbus summary: %s", text)
			}
		})
	}
}

func TestAutoSaverWritesAndSplitsBySize(t *testing.T) {
	dir := t.TempDir()
	saver, err := newAutoSaver("m1", AutoSaveOptions{
		Enabled:     true,
		Directory:   dir,
		BaseName:    "capture",
		Format:      ExportText,
		SplitMode:   SplitSize,
		SplitSizeKB: 1,
	})
	if err != nil {
		t.Fatalf("newAutoSaver: %v", err)
	}
	frame := sampleFrames()[0]
	for i := 0; i < 80; i++ {
		frame.Seq = int64(i + 1)
		if err := saver.write(frame); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if err := saver.close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "capture-*.txt"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("autosave files = %d, want at least 2", len(files))
	}
}
