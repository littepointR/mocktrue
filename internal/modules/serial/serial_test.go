package serial

import (
	"context"
	"testing"

	mterrors "github.com/suyue/mocktrue/internal/core/errors"
)

func TestIDAndManifest(t *testing.T) {
	t.Parallel()
	m := New()
	if m.ID() != "serial" {
		t.Fatalf("ID = %q, want serial", m.ID())
	}
	mf := m.Manifest()
	if mf.ID != "serial" || mf.Version == "" {
		t.Fatalf("Manifest = %+v, want ID serial and non-empty version", mf)
	}
	if mf.Frontend.ActivityIcon != "serial" || mf.Frontend.ActivityTitle == "" {
		t.Fatalf("Frontend contribution missing: %+v", mf.Frontend)
	}
	if len(mf.Frontend.Views) == 0 {
		t.Fatalf("Manifest must contribute at least one view")
	}
}

func TestServicesWrappedReturnsOneService(t *testing.T) {
	t.Parallel()
	m := New()
	if err := m.Init(context.Background(), testDeps()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	wrapped := m.ServicesWrapped()
	if len(wrapped) != 1 {
		t.Fatalf("ServicesWrapped len = %d, want 1", len(wrapped))
	}
	if wrapped[0].Instance() == nil {
		t.Fatalf("wrapped service instance must not be nil")
	}
	if _, ok := wrapped[0].Instance().(*Service); !ok {
		t.Fatalf("wrapped instance must be *serial.Service, got %T", wrapped[0].Instance())
	}
}

func TestPingEmptyReturnsInvalid(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	_, err := svc.Ping(context.Background(), "")
	if err == nil {
		t.Fatalf("Ping(\"\") must error")
	}
	if mterrors.AsCode(err) != mterrors.CodeInvalid {
		t.Fatalf("Ping(\"\") code = %q, want invalid", mterrors.AsCode(err))
	}
}

func TestPingReturnsPong(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	got, err := svc.Ping(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if got != "pong:hi" {
		t.Fatalf("Ping = %q, want pong:hi", got)
	}
}

func TestLifecycleDoesNotPanicBeforeInit(t *testing.T) {
	t.Parallel()
	m := New()
	// Stop/Dispose must be safe even if Init was never called (rollback path).
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop before Init must not error: %v", err)
	}
	m.Dispose()
	m.Dispose() // idempotent
}

func TestStartStopAfterInit(t *testing.T) {
	t.Parallel()
	m := New()
	deps := testDeps()
	if err := m.Init(context.Background(), deps); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	m.Dispose()
}
