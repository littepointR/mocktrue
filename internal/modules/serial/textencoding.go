package serial

import (
	"fmt"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	textunicode "golang.org/x/text/encoding/unicode"
)

func encodeSerialText(content string, label string) ([]byte, error) {
	normalized := normalizeSerialTextEncoding(label)
	switch normalized {
	case "", "utf-8", "utf8":
		return []byte(content), nil
	case "ascii":
		return encodeASCII(content)
	case "utf-16le":
		return encodeWith(content, textunicode.UTF16(textunicode.LittleEndian, textunicode.IgnoreBOM))
	case "utf-16be":
		return encodeWith(content, textunicode.UTF16(textunicode.BigEndian, textunicode.IgnoreBOM))
	case "gb2312", "gbk":
		return encodeWith(content, simplifiedchinese.GBK)
	case "gb18030":
		return encodeWith(content, simplifiedchinese.GB18030)
	case "big5":
		return encodeWith(content, traditionalchinese.Big5)
	case "shift-jis", "shiftjis":
		return encodeWith(content, japanese.ShiftJIS)
	case "windows-1251", "windows1251", "cp1251":
		return encodeWith(content, charmap.Windows1251)
	case "windows-1252", "windows1252", "cp1252":
		return encodeWith(content, charmap.Windows1252)
	default:
		return nil, fmt.Errorf("unsupported text encoding: %s", label)
	}
}

func decodeSerialText(data []byte, label string) (string, error) {
	normalized := normalizeSerialTextEncoding(label)
	switch normalized {
	case "", "utf-8", "utf8":
		return string(data), nil
	case "ascii":
		return decodeASCII(data), nil
	case "utf-16le":
		return decodeWith(data, textunicode.UTF16(textunicode.LittleEndian, textunicode.IgnoreBOM))
	case "utf-16be":
		return decodeWith(data, textunicode.UTF16(textunicode.BigEndian, textunicode.IgnoreBOM))
	case "gb2312", "gbk":
		return decodeWith(data, simplifiedchinese.GBK)
	case "gb18030":
		return decodeWith(data, simplifiedchinese.GB18030)
	case "big5":
		return decodeWith(data, traditionalchinese.Big5)
	case "shift-jis", "shiftjis":
		return decodeWith(data, japanese.ShiftJIS)
	case "windows-1251", "windows1251", "cp1251":
		return decodeWith(data, charmap.Windows1251)
	case "windows-1252", "windows1252", "cp1252":
		return decodeWith(data, charmap.Windows1252)
	default:
		return "", fmt.Errorf("unsupported text encoding: %s", label)
	}
}

func normalizeSerialTextEncoding(label string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(label)), "_", "-")
}

func encodeASCII(content string) ([]byte, error) {
	out := make([]byte, 0, len(content))
	for _, r := range content {
		if r > 0x7f {
			return nil, fmt.Errorf("rune %q is not supported by ascii", r)
		}
		out = append(out, byte(r))
	}
	return out, nil
}

func decodeASCII(data []byte) string {
	out := make([]rune, 0, len(data))
	for _, b := range data {
		if b > 0x7f {
			out = append(out, '\ufffd')
			continue
		}
		out = append(out, rune(b))
	}
	return string(out)
}

func encodeWith(content string, enc encoding.Encoding) ([]byte, error) {
	return enc.NewEncoder().Bytes([]byte(content))
}

func decodeWith(data []byte, enc encoding.Encoding) (string, error) {
	decoded, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
