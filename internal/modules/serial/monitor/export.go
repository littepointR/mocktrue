package monitor

import (
	"encoding/csv"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/suyue/mocktrue/internal/core/errors"
	"github.com/suyue/mocktrue/internal/modules/serial/recorder"
)

func exportFrames(req ExportRequest, frames []Frame) (string, error) {
	if req.Path == "" {
		return "", errors.New(errors.CodeInvalid, "export path must not be empty")
	}
	format := normalizeExportFormat(req.Format)
	if err := os.MkdirAll(filepath.Dir(req.Path), 0o700); err != nil {
		return "", errors.Wrap(errors.CodeIO, "create export directory", err)
	}
	switch format {
	case ExportCSV:
		return req.Path, writeCSV(req.Path, frames)
	case ExportHTML:
		return req.Path, writeHTML(req.Path, frames)
	case ExportPCAP:
		return req.Path, writePCAP(req.MonitorID, req.Path, frames)
	default:
		return req.Path, writeText(req.Path, frames)
	}
}

func normalizeExportFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case ExportCSV:
		return ExportCSV
	case ExportHTML:
		return ExportHTML
	case ExportPCAP, "pcap":
		return ExportPCAP
	default:
		return ExportText
	}
}

func writeCSV(path string, frames []Frame) error {
	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(errors.CodeIO, "create csv export", err)
	}
	defer file.Close()
	w := csv.NewWriter(file)
	if err := w.Write([]string{"seq", "timestamp", "direction", "port", "length", "hex", "text", "modbus"}); err != nil {
		return errors.Wrap(errors.CodeIO, "write csv header", err)
	}
	for _, frame := range frames {
		modbus := ""
		if frame.Modbus != nil {
			modbus = frame.Modbus.Summary
			if frame.Modbus.Error != "" {
				modbus += " " + frame.Modbus.Error
			}
		}
		if err := w.Write([]string{
			fmt.Sprintf("%d", frame.Seq),
			frame.Timestamp.Format("2006-01-02 15:04:05.000"),
			frame.Direction,
			frame.Port,
			fmt.Sprintf("%d", frame.Length),
			frame.DisplayHex,
			frame.DisplayText,
			modbus,
		}); err != nil {
			return errors.Wrap(errors.CodeIO, "write csv row", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return errors.Wrap(errors.CodeIO, "flush csv export", err)
	}
	return nil
}

func writeText(path string, frames []Frame) error {
	var b strings.Builder
	for _, frame := range frames {
		fmt.Fprintf(&b, "%06d %s %-6s %-24s len=%d hex=%s text=%s",
			frame.Seq,
			frame.Timestamp.Format("2006-01-02 15:04:05.000"),
			frame.Direction,
			frame.Port,
			frame.Length,
			frame.DisplayHex,
			escapeLineBreaks(frame.DisplayText),
		)
		if frame.Modbus != nil {
			fmt.Fprintf(&b, " modbus=%s", frame.Modbus.Summary)
			if frame.Modbus.Error != "" {
				fmt.Fprintf(&b, " error=%s", frame.Modbus.Error)
			}
		}
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return errors.Wrap(errors.CodeIO, "write text export", err)
	}
	return nil
}

func writeHTML(path string, frames []Frame) error {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>MockTrue 串口监控</title>")
	b.WriteString("<style>body{font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;margin:16px}table{border-collapse:collapse;width:100%;font-size:12px}th,td{border:1px solid #ddd;padding:4px 6px;text-align:left;vertical-align:top}th{background:#f2f2f2}code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}</style>")
	b.WriteString("</head><body><h1>MockTrue 串口监控</h1><table><thead><tr><th>Seq</th><th>Time</th><th>Direction</th><th>Port</th><th>Len</th><th>Hex</th><th>Text</th><th>Modbus</th></tr></thead><tbody>")
	for _, frame := range frames {
		modbus := ""
		if frame.Modbus != nil {
			modbus = frame.Modbus.Summary
			if frame.Modbus.Error != "" {
				modbus += " " + frame.Modbus.Error
			}
		}
		fmt.Fprintf(&b, "<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td></tr>",
			frame.Seq,
			html.EscapeString(frame.Timestamp.Format("2006-01-02 15:04:05.000")),
			html.EscapeString(frame.Direction),
			html.EscapeString(frame.Port),
			frame.Length,
			html.EscapeString(frame.DisplayHex),
			html.EscapeString(frame.DisplayText),
			html.EscapeString(modbus),
		)
	}
	b.WriteString("</tbody></table></body></html>")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return errors.Wrap(errors.CodeIO, "write html export", err)
	}
	return nil
}

func writePCAP(monitorID, path string, frames []Frame) error {
	rec, err := recorder.NewRecorder(monitorID, path)
	if err != nil {
		return err
	}
	defer rec.Close()
	for _, frame := range frames {
		dir := recorder.DirectionRX
		if frame.Direction == DirectionAToB {
			dir = recorder.DirectionTX
		}
		if err := rec.WriteFrame(dir, frame.Timestamp, frame.Data); err != nil {
			return err
		}
	}
	return nil
}

func escapeLineBreaks(value string) string {
	return strings.NewReplacer("\r", "\\r", "\n", "\\n").Replace(value)
}
