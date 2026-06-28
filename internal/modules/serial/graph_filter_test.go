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
		NodeType:  "serial.virtual",
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
		"len > 0 or",
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

func TestSerialGraphFilterExpressionCoversFallbackFieldsComparisonsOrAndEscapes(t *testing.T) {
	candidate := serialGraphFilterCandidate{
		ByteLength:  4,
		ByteCount:   9,
		PayloadText: "Fallback Text",
		PayloadHex:  "0D 0A",
		Message:     "accepted",
		Level:       "info",
		Source:      "bus",
		GraphID:     "g",
		NodeID:      "n",
		NodeType:    "serial.filter",
		Direction:   "rx",
		Details:     "line	value",
		Action:      "forward",
		Category:    "serial.graph",
	}

	tests := []struct {
		name       string
		expression string
		want       bool
	}{
		{name: "byte length or comparison", expression: `byteLength <= 4 or nodeID == "missing"`, want: true},
		{name: "byte count numeric operators", expression: `byteCount != 8 and byteCount > 3 and byteCount >= 4 and byteCount < 10`, want: true},
		{name: "payload text fallback", expression: `text == "fallback text"`, want: true},
		{name: "payload hex fallback normalizes", expression: `hex == "0d0a"`, want: true},
		{name: "escaped tab string", expression: `details contains "line	value"`, want: true},
		{name: "action and category", expression: `action == "forward" and category != "other"`, want: true},
		{name: "metadata fields", expression: `source == "bus" and graphid == "g" and nodeid == "n" and nodetype == "serial.filter" and direction == "rx"`, want: true},
		{name: "payload fields", expression: `payloadtext contains "text" and payloadhex contains "0d 0a"`, want: true},
		{name: "case-sensitive fallback mismatch", expression: `text == "fallback text"`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchSerialGraphFilter(candidate, serialGraphFilterOptions{
				Mode:          "expression",
				Expression:    tt.expression,
				CaseSensitive: tt.name == "case-sensitive fallback mismatch",
			})
			if err != nil {
				t.Fatalf("matchSerialGraphFilter returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("matchSerialGraphFilter = %v, want %v", got, tt.want)
			}
		})
	}
	if got := (serialGraphFilterCandidate{ByteCount: 9}).length(); got != 9 {
		t.Fatalf("ByteCount-only length = %d, want 9", got)
	}
}

func TestSerialGraphFilterStringScannerParserAndComparisonFallbacks(t *testing.T) {
	input := `"quote: \" slash: \\ newline: \n carriage: \r tab: 	 other: \x"`
	got, next, err := scanSerialGraphFilterString(input, 0)
	if err != nil {
		t.Fatalf("scanSerialGraphFilterString returned error: %v", err)
	}
	want := "quote: \" slash: \\ newline: \n carriage: \r tab: 	 other: x"
	if got != want || next != len(input) {
		t.Fatalf("scanSerialGraphFilterString = %q, %d; want %q, %d", got, next, want, len(input))
	}
	if _, _, err := scanSerialGraphFilterString(`"unterminated\`, 0); err == nil {
		t.Fatalf("scanSerialGraphFilterString must reject unterminated escape")
	}
	if _, err := tokenizeSerialGraphFilterExpression(`text == "unterminated`); err == nil {
		t.Fatalf("tokenizeSerialGraphFilterExpression must reject unterminated string")
	}

	parser := serialGraphFilterParser{position: 1}
	if token := parser.peek(); token.kind != serialGraphFilterTokenEOF {
		t.Fatalf("peek past end = %#v, want EOF", token)
	}
	if isSerialGraphNumericOperator("contains") {
		t.Fatalf("contains must not be a numeric operator")
	}
	if compareSerialGraphFilterNumber(1, "contains", 1) {
		t.Fatalf("unknown numeric comparison operator must not match")
	}
	if compareSerialGraphFilterString("text", "left", "matches", "left", serialGraphFilterOptions{}) {
		t.Fatalf("unknown string comparison operator must not match")
	}
}
