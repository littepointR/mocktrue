package recorder

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRecorderWriteAndPlayback(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.pcapng")

	// Record
	rec, err := NewRecorder("test-port", filePath)
	if err != nil {
		t.Fatalf("NewRecorder failed: %v", err)
	}

	now := time.Now()
	data1 := []byte("hello")
	data2 := []byte("world")

	if err := rec.WriteFrame(DirectionRX, now, data1); err != nil {
		t.Fatalf("WriteFrame RX failed: %v", err)
	}
	if err := rec.WriteFrame(DirectionTX, now.Add(time.Millisecond), data2); err != nil {
		t.Fatalf("WriteFrame TX failed: %v", err)
	}

	if rec.FrameCount() != 2 {
		t.Fatalf("FrameCount = %d, want 2", rec.FrameCount())
	}

	if err := rec.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Playback
	pb, err := NewPlayback(filePath)
	if err != nil {
		t.Fatalf("NewPlayback failed: %v", err)
	}
	defer pb.Close()

	frames, err := pb.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames len = %d, want 2", len(frames))
	}

	// Verify frame 1
	if frames[0].Direction != DirectionRX {
		t.Fatalf("frame[0] direction = %d, want RX(0)", frames[0].Direction)
	}
	if string(frames[0].Data) != "hello" {
		t.Fatalf("frame[0] data = %q, want hello", frames[0].Data)
	}

	// Verify frame 2
	if frames[1].Direction != DirectionTX {
		t.Fatalf("frame[1] direction = %d, want TX(1)", frames[1].Direction)
	}
	if string(frames[1].Data) != "world" {
		t.Fatalf("frame[1] data = %q, want world", frames[1].Data)
	}
}

func TestRecorderEmptyPathRejects(t *testing.T) {
	t.Parallel()
	_, err := NewRecorder("port", "")
	if err == nil {
		t.Fatalf("empty path must error")
	}
}

func TestPlaybackEmptyPathRejects(t *testing.T) {
	t.Parallel()
	_, err := NewPlayback("")
	if err == nil {
		t.Fatalf("empty path must error")
	}
}
