package recorder

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/suyue/mocktrue/internal/core/errors"
)

const (
	// DLT_USER0 is the custom link type for serial port data.
	DLT_USER0 = 147

	// DirectionRX indicates received data.
	DirectionRX byte = 0
	// DirectionTX indicates transmitted data.
	DirectionTX byte = 1
)

// Frame represents a recorded serial frame.
type Frame struct {
	Direction byte
	Timestamp time.Time
	Data      []byte
}

// Recorder writes serial frames to a pcap-ng file.
type Recorder struct {
	mu       sync.Mutex
	writer   *pcapgo.NgWriter
	file     *os.File
	portID   string
	isOpen   bool
	frameCnt int64
}

// NewRecorder creates a new recorder for the given port.
func NewRecorder(portID, filePath string) (*Recorder, error) {
	if filePath == "" {
		return nil, errors.New(errors.CodeInvalid, "file path must not be empty")
	}

	f, err := os.Create(filePath)
	if err != nil {
		return nil, errors.Wrap(errors.CodeIO, fmt.Sprintf("create pcap file %s", filePath), err)
	}

	w, err := pcapgo.NewNgWriter(f, layers.LinkType(DLT_USER0))
	if err != nil {
		f.Close()
		return nil, errors.Wrap(errors.CodeIO, "create pcap-ng writer", err)
	}

	return &Recorder{
		writer: w,
		file:   f,
		portID: portID,
		isOpen: true,
	}, nil
}

// WriteFrame records a frame with direction and timestamp.
func (r *Recorder) WriteFrame(direction byte, ts time.Time, data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isOpen {
		return errors.New(errors.CodeInvalid, "recorder is not open")
	}

	// Build custom frame: [1B direction][4B length][payload]
	frameData := make([]byte, 5+len(data))
	frameData[0] = direction
	binary.BigEndian.PutUint32(frameData[1:5], uint32(len(data)))
	copy(frameData[5:], data)

	ci := gopacket.CaptureInfo{
		Timestamp:     ts,
		CaptureLength: len(frameData),
		Length:        len(frameData),
	}

	if err := r.writer.WritePacket(ci, frameData); err != nil {
		return errors.Wrap(errors.CodeIO, "write pcap packet", err)
	}

	r.frameCnt++
	return nil
}

// Close flushes and closes the recorder.
func (r *Recorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isOpen {
		return nil
	}

	r.isOpen = false
	if err := r.writer.Flush(); err != nil {
		r.file.Close()
		return errors.Wrap(errors.CodeIO, "flush pcap", err)
	}
	return r.file.Close()
}

// FrameCount returns the number of recorded frames.
func (r *Recorder) FrameCount() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.frameCnt
}
