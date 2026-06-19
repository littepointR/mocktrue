package monitor

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/suyue/mocktrue/internal/core/errors"
	"github.com/suyue/mocktrue/internal/modules/serial/recorder"
)

type autoSaver struct {
	mu        sync.Mutex
	monitorID string
	options   AutoSaveOptions
	createdAt time.Time
	part      int
	written   int64
	file      *os.File
	csv       *csvLineWriter
	pcap      *recorder.Recorder
}

func newAutoSaver(monitorID string, options AutoSaveOptions) (*autoSaver, error) {
	normalized := normalizeAutoSaveOptions(options)
	if !normalized.Enabled {
		return nil, nil
	}
	saver := &autoSaver{monitorID: monitorID, options: normalized}
	if err := saver.rotateLocked(time.Now()); err != nil {
		return nil, err
	}
	return saver, nil
}

func normalizeAutoSaveOptions(options AutoSaveOptions) AutoSaveOptions {
	options.Format = normalizeExportFormat(options.Format)
	if options.SplitMode == "" {
		options.SplitMode = SplitNone
	}
	if options.BaseName == "" {
		options.BaseName = "serial-monitor"
	}
	if options.Encoding == "" {
		options.Encoding = "utf-8"
	}
	return options
}

func (s *autoSaver) write(frame Frame) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.rotateIfNeededLocked(time.Now()); err != nil {
		return err
	}
	before := s.written
	switch s.options.Format {
	case ExportCSV:
		if err := s.csv.write(frame); err != nil {
			return err
		}
	case ExportHTML:
		if _, err := s.file.WriteString(htmlRow(frame)); err != nil {
			return errors.Wrap(errors.CodeIO, "write autosave html", err)
		}
	case ExportPCAP:
		dir := recorder.DirectionRX
		if frame.Direction == DirectionAToB {
			dir = recorder.DirectionTX
		}
		if err := s.pcap.WriteFrame(dir, frame.Timestamp, frame.Data); err != nil {
			return err
		}
	default:
		if _, err := s.file.WriteString(textLine(frame)); err != nil {
			return errors.Wrap(errors.CodeIO, "write autosave text", err)
		}
	}
	if s.file != nil {
		if info, err := s.file.Stat(); err == nil {
			s.written = info.Size()
		}
	}
	if s.written == before {
		s.written += int64(len(frame.Data))
	}
	return nil
}

func (s *autoSaver) close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeLocked()
}

func (s *autoSaver) rotateIfNeededLocked(now time.Time) error {
	switch s.options.SplitMode {
	case SplitSize:
		limit := int64(s.options.SplitSizeKB) * 1024
		if limit > 0 && s.written >= limit {
			return s.rotateLocked(now)
		}
	case SplitTime:
		limit := time.Duration(s.options.SplitIntervalSeconds) * time.Second
		if limit > 0 && !s.createdAt.IsZero() && now.Sub(s.createdAt) >= limit {
			return s.rotateLocked(now)
		}
	}
	return nil
}

func (s *autoSaver) rotateLocked(now time.Time) error {
	if err := s.closeLocked(); err != nil {
		return err
	}
	s.part++
	s.createdAt = now
	s.written = 0

	path := s.pathLocked(now)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return errors.Wrap(errors.CodeIO, "create autosave directory", err)
	}
	if s.options.Format == ExportPCAP {
		rec, err := recorder.NewRecorder(s.monitorID, path)
		if err != nil {
			return err
		}
		s.pcap = rec
		return nil
	}

	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(errors.CodeIO, "create autosave file", err)
	}
	s.file = file
	switch s.options.Format {
	case ExportCSV:
		s.csv = &csvLineWriter{file: file}
		return s.csv.header()
	case ExportHTML:
		_, err = file.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>MockTrue 串口监控</title></head><body><table><thead><tr><th>Seq</th><th>Time</th><th>Direction</th><th>Port</th><th>Len</th><th>Hex</th><th>Text</th><th>Modbus</th></tr></thead><tbody>\n")
		if err != nil {
			return errors.Wrap(errors.CodeIO, "write autosave html header", err)
		}
		return nil
	default:
		return nil
	}
}

func (s *autoSaver) pathLocked(now time.Time) string {
	if s.options.Path != "" && s.options.SplitMode == SplitNone {
		return s.options.Path
	}
	dir := s.options.Directory
	if dir == "" && s.options.Path != "" {
		dir = filepath.Dir(s.options.Path)
	}
	if dir == "" {
		dir = "."
	}
	base := strings.TrimSuffix(s.options.BaseName, filepath.Ext(s.options.BaseName))
	ext := "." + s.options.Format
	if s.options.Format == ExportText {
		ext = ".txt"
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%s-%03d%s", base, now.Format("20060102-150405"), s.part, ext))
}

func (s *autoSaver) closeLocked() error {
	if s.pcap != nil {
		err := s.pcap.Close()
		s.pcap = nil
		return err
	}
	if s.file == nil {
		return nil
	}
	if s.options.Format == ExportHTML {
		if _, err := s.file.WriteString("</tbody></table></body></html>"); err != nil {
			_ = s.file.Close()
			s.file = nil
			return errors.Wrap(errors.CodeIO, "write autosave html footer", err)
		}
	}
	err := s.file.Close()
	s.file = nil
	s.csv = nil
	if err != nil {
		return errors.Wrap(errors.CodeIO, "close autosave file", err)
	}
	return nil
}

type csvLineWriter struct {
	file *os.File
}

func (w *csvLineWriter) header() error {
	_, err := w.file.WriteString("seq,timestamp,direction,port,length,hex,text,modbus\n")
	if err != nil {
		return errors.Wrap(errors.CodeIO, "write autosave csv header", err)
	}
	return nil
}

func (w *csvLineWriter) write(frame Frame) error {
	line := []string{
		fmt.Sprintf("%d", frame.Seq),
		frame.Timestamp.Format("2006-01-02 15:04:05.000"),
		frame.Direction,
		frame.Port,
		fmt.Sprintf("%d", frame.Length),
		frame.DisplayHex,
		frame.DisplayText,
		modbusText(frame),
	}
	escaped := make([]string, 0, len(line))
	for _, value := range line {
		escaped = append(escaped, csvEscape(value))
	}
	_, err := w.file.WriteString(strings.Join(escaped, ",") + "\n")
	if err != nil {
		return errors.Wrap(errors.CodeIO, "write autosave csv row", err)
	}
	return nil
}

func csvEscape(value string) string {
	if strings.ContainsAny(value, "\",\r\n") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return value
}

func textLine(frame Frame) string {
	return fmt.Sprintf("%06d %s %-6s %-24s len=%d hex=%s text=%s modbus=%s\n",
		frame.Seq,
		frame.Timestamp.Format("2006-01-02 15:04:05.000"),
		frame.Direction,
		frame.Port,
		frame.Length,
		frame.DisplayHex,
		escapeLineBreaks(frame.DisplayText),
		modbusText(frame),
	)
}

func htmlRow(frame Frame) string {
	return fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td></tr>\n",
		frame.Seq,
		html.EscapeString(frame.Timestamp.Format("2006-01-02 15:04:05.000")),
		html.EscapeString(frame.Direction),
		html.EscapeString(frame.Port),
		frame.Length,
		html.EscapeString(frame.DisplayHex),
		html.EscapeString(frame.DisplayText),
		html.EscapeString(modbusText(frame)),
	)
}

func modbusText(frame Frame) string {
	if frame.Modbus == nil {
		return ""
	}
	if frame.Modbus.Error != "" {
		return frame.Modbus.Summary + " " + frame.Modbus.Error
	}
	return frame.Modbus.Summary
}
