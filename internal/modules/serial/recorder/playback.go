package recorder

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/google/gopacket/pcapgo"
	"github.com/littepointR/portweave/internal/core/errors"
)

// Playback reads frames from a pcap-ng file.
type Playback struct {
	reader *pcapgo.NgReader
	file   *os.File
}

// NewPlayback opens a pcap-ng file for reading.
func NewPlayback(filePath string) (*Playback, error) {
	if filePath == "" {
		return nil, errors.New(errors.CodeInvalid, "file path must not be empty")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrap(errors.CodeIO, fmt.Sprintf("open pcap file %s", filePath), err)
	}

	r, err := pcapgo.NewNgReader(f, pcapgo.DefaultNgReaderOptions)
	if err != nil {
		_ = f.Close()
		return nil, errors.Wrap(errors.CodeIO, "create pcap-ng reader", err)
	}

	return &Playback{
		reader: r,
		file:   f,
	}, nil
}

// ReadFrame reads the next frame from the pcap file. Returns io.EOF when
// no more frames are available.
func (p *Playback) ReadFrame() (Frame, error) {
	data, ci, err := p.reader.ReadPacketData()
	if err != nil {
		if err == io.EOF {
			return Frame{}, io.EOF
		}
		return Frame{}, errors.Wrap(errors.CodeIO, "read pcap packet", err)
	}

	if len(data) < 5 {
		return Frame{}, errors.New(errors.CodeInvalid, "pcap frame too short")
	}

	direction := data[0]
	length := binary.BigEndian.Uint32(data[1:5])
	if int(length) > len(data)-5 {
		length = uint32(len(data) - 5)
	}

	return Frame{
		Direction: direction,
		Timestamp: ci.Timestamp,
		Data:      data[5 : 5+length],
	}, nil
}

// Close closes the playback file.
func (p *Playback) Close() error {
	return p.file.Close()
}

// ReadAll reads all frames from the file.
func (p *Playback) ReadAll() ([]Frame, error) {
	var frames []Frame
	for {
		frame, err := p.ReadFrame()
		if err == io.EOF {
			break
		}
		if err != nil {
			return frames, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
