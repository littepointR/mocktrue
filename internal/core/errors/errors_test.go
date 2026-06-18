package errors

import (
	stderrors "errors"
	"testing"
)

func TestNewRequiresNonEmptyCodeAndMessage(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("New with empty code must panic")
		}
	}()
	_ = New("", "msg")
}

func TestNewEmptyMessagePanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("New with empty message must panic")
		}
	}()
	_ = New(CodeInvalid, "")
}

func TestNewErrorText(t *testing.T) {
	t.Parallel()
	e := New(CodeNotFound, "port not found")
	if e.Code != CodeNotFound {
		t.Fatalf("Code = %q, want %q", e.Code, CodeNotFound)
	}
	if e.Error() != "not_found: port not found" {
		t.Fatalf("Error() = %q, want %q", e.Error(), "not_found: port not found")
	}
	if e.Unwrap() != nil {
		t.Fatalf("Unwrap of New error should be nil, got %v", e.Unwrap())
	}
}

func TestWrapRequiresNonNilCause(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Wrap with nil cause must panic")
		}
	}()
	_ = Wrap(CodeIO, "write failed", nil)
}

func TestWrapPreservesCause(t *testing.T) {
	t.Parallel()
	cause := stderrors.New("disk full")
	e := Wrap(CodeIO, "write failed", cause)
	if e.Unwrap() == nil || e.Unwrap().Error() != "disk full" {
		t.Fatalf("Unwrap = %v, want disk full", e.Unwrap())
	}
	want := "io: write failed: disk full"
	if e.Error() != want {
		t.Fatalf("Error() = %q, want %q", e.Error(), want)
	}
}

func TestWithFieldIsImmutable(t *testing.T) {
	t.Parallel()
	orig := New(CodeInvalid, "bad input")
	mutated := orig.WithField("port", "COM3")

	if orig.Fields != nil {
		t.Fatalf("original must keep nil Fields, got %v", orig.Fields)
	}
	if mutated.Fields["port"] != "COM3" {
		t.Fatalf("mutated Fields[port] = %v, want COM3", mutated.Fields["port"])
	}
	if &orig.Fields == &mutated.Fields {
		t.Fatalf("WithField must not share the Fields map reference")
	}
}

func TestWithFieldChains(t *testing.T) {
	t.Parallel()
	e := New(CodeInternal, "oops").
		WithField("module", "serial").
		WithField("port", "COM3")
	if e.Fields["module"] != "serial" || e.Fields["port"] != "COM3" {
		t.Fatalf("chained fields lost: %v", e.Fields)
	}
}

func TestAsCodeExtractsFromWrappedError(t *testing.T) {
	t.Parallel()
	wrapped := Wrap(CodeNotFound, "missing", stderrors.New("enoent"))
	if got := AsCode(wrapped); got != CodeNotFound {
		t.Fatalf("AsCode(wrapped) = %q, want %q", got, CodeNotFound)
	}
}

func TestAsCodeDefaultsToInternalForPlainError(t *testing.T) {
	t.Parallel()
	if got := AsCode(stderrors.New("plain")); got != CodeInternal {
		t.Fatalf("AsCode(plain) = %q, want %q", got, CodeInternal)
	}
}

func TestAsCodeHandlesNil(t *testing.T) {
	t.Parallel()
	if got := AsCode(nil); got != CodeInternal {
		t.Fatalf("AsCode(nil) = %q, want %q", got, CodeInternal)
	}
}
