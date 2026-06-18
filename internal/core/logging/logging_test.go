package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestDefaultNonNil(t *testing.T) {
	t.Parallel()
	if Default() == nil {
		t.Fatalf("Default() must not return nil")
	}
}

func TestNamedReturnsNewInstanceAndLeavesOriginal(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := New(slog.NewTextHandler(&buf, nil))
	named := base.Named("serial")

	if named == base {
		t.Fatalf("Named must return a distinct Logger")
	}

	named.Info("hello")
	out := buf.String()
	if !strings.Contains(out, "module=serial") {
		t.Fatalf("named log output missing module=serial: %q", out)
	}
}

func TestNamedEmptyDoesNotPanicAndLogs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := New(slog.NewTextHandler(&buf, nil))
	named := base.Named("")
	if named == nil {
		t.Fatalf("Named(\"\") must not return nil")
	}
}

func TestLevelFiltering(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	logger := New(handler)
	logger.Info("should be filtered")
	logger.Warn("should appear")

	out := buf.String()
	if strings.Contains(out, "should be filtered") {
		t.Fatalf("Info should be filtered at Warn level: %q", out)
	}
	if !strings.Contains(out, "should appear") {
		t.Fatalf("Warn should appear: %q", out)
	}
}
