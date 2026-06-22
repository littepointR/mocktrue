package serial

import (
	"strings"
	"testing"
)

func TestSerialGraphFilterPlainHonorsCaseAndWholeWord(t *testing.T) {
	candidate := serialGraphFilterCandidate{
		Text:    "Status OK on reporter",
		Hex:     "4f 4b",
		Message: "Gateway ready",
		Level:   "info",
	}

	tests := []struct {
		name    string
		options serialGraphFilterOptions
		want    bool
	}{
		{
			name:    "empty expression matches",
			options: serialGraphFilterOptions{Mode: "plain", Expression: ""},
			want:    true,
		},
		{
			name:    "case insensitive plain keyword matches",
			options: serialGraphFilterOptions{Mode: "plain", Expression: "ok"},
			want:    true,
		},
		{
			name:    "case sensitive plain keyword rejects different case",
			options: serialGraphFilterOptions{Mode: "plain", Expression: "ok", CaseSensitive: true},
			want:    false,
		},
		{
			name:    "non whole word matches substring",
			options: serialGraphFilterOptions{Mode: "plain", Expression: "port"},
			want:    true,
		},
		{
			name:    "whole word rejects substring",
			options: serialGraphFilterOptions{Mode: "plain", Expression: "port", WholeWord: true},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchSerialGraphFilter(candidate, tt.options)
			if err != nil {
				t.Fatalf("matchSerialGraphFilter returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("matchSerialGraphFilter = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSerialGraphFilterRegexHonorsCaseWholeWordAndErrors(t *testing.T) {
	candidate := serialGraphFilterCandidate{Text: "TEMP=42 error", Message: "payload accepted"}

	tests := []struct {
		name    string
		options serialGraphFilterOptions
		want    bool
	}{
		{
			name:    "case insensitive regex matches",
			options: serialGraphFilterOptions{Mode: "regex", Expression: `temp=\d+`},
			want:    true,
		},
		{
			name:    "case sensitive regex rejects different case",
			options: serialGraphFilterOptions{Mode: "regex", Expression: `temp=\d+`, CaseSensitive: true},
			want:    false,
		},
		{
			name:    "non whole word regex matches substring",
			options: serialGraphFilterOptions{Mode: "regex", Expression: `err`},
			want:    true,
		},
		{
			name:    "whole word regex rejects substring",
			options: serialGraphFilterOptions{Mode: "regex", Expression: `err`, WholeWord: true},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchSerialGraphFilter(candidate, tt.options)
			if err != nil {
				t.Fatalf("matchSerialGraphFilter returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("matchSerialGraphFilter = %v, want %v", got, tt.want)
			}
		})
	}

	_, err := matchSerialGraphFilter(candidate, serialGraphFilterOptions{Mode: "regex", Expression: "[unterminated"})
	if err == nil || !strings.Contains(err.Error(), "invalid regex") {
		t.Fatalf("invalid regex error = %v, want invalid regex", err)
	}
}

func TestSerialGraphFilterExpressionSubset(t *testing.T) {
	candidate := serialGraphFilterCandidate{
		Len:       7,
		Text:      "TEMP OK",
		Hex:       "0d 0a ff",
		Message:   "template failed with fallback",
		Level:     "error",
		Source:    "runtime",
		GraphID:   "graph-1",
		NodeID:    "rx-1",
		NodeType:  "serial.receiver",
		Direction: "rx",
	}

	tests := []struct {
		name    string
		options serialGraphFilterOptions
		want    bool
	}{
		{name: "length comparison", options: serialGraphFilterOptions{Mode: "expression", Expression: "len > 0"}, want: true},
		{name: "text contains", options: serialGraphFilterOptions{Mode: "expression", Expression: `text contains "OK"`}, want: true},
		{name: "hex contains normalizes whitespace", options: serialGraphFilterOptions{Mode: "expression", Expression: `hex contains "0d0a"`}, want: true},
		{name: "and expression", options: serialGraphFilterOptions{Mode: "expression", Expression: `len >= 4 and text contains "TEMP"`}, want: true},
		{name: "not expression", options: serialGraphFilterOptions{Mode: "expression", Expression: `not (hex contains "ab")`}, want: true},
		{name: "log fields", options: serialGraphFilterOptions{Mode: "expression", Expression: `level == "error" and message contains "template"`}, want: true},
		{name: "case sensitive string operation", options: serialGraphFilterOptions{Mode: "expression", Expression: `text contains "ok"`, CaseSensitive: true}, want: false},
		{name: "case insensitive string operation", options: serialGraphFilterOptions{Mode: "expression", Expression: `text contains "ok"`}, want: true},
		{name: "whole word ignored in expression mode", options: serialGraphFilterOptions{Mode: "expression", Expression: `text contains "EMP"`, WholeWord: true}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchSerialGraphFilter(candidate, tt.options)
			if err != nil {
				t.Fatalf("matchSerialGraphFilter returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("matchSerialGraphFilter = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSerialGraphFilterExpressionInvalidErrors(t *testing.T) {
	candidate := serialGraphFilterCandidate{Len: 3, Text: "abc"}
	tests := []string{
		"len >> 2",
		"text contains",
		"(len > 1",
		`unknown contains "x"`,
	}

	for _, expression := range tests {
		t.Run(expression, func(t *testing.T) {
			_, err := matchSerialGraphFilter(candidate, serialGraphFilterOptions{Mode: "expression", Expression: expression})
			if err == nil {
				t.Fatalf("matchSerialGraphFilter(%q) error = nil, want structured error", expression)
			}
		})
	}
}
