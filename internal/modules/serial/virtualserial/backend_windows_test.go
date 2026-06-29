//go:build windows

package virtualserial

import (
	"testing"

	coreerrors "github.com/littepointR/portweave/internal/core/errors"
)

func TestNormalizeWindowsPortName(t *testing.T) {
	got, err := normalizeWindowsPortName(`\\.\com12`)
	if err != nil {
		t.Fatalf("normalizeWindowsPortName: %v", err)
	}
	if got != "COM12" {
		t.Fatalf("normalized = %q, want COM12", got)
	}

	if _, err := normalizeWindowsPortName("ttyV0"); coreerrors.AsCode(err) != coreerrors.CodeInvalid {
		t.Fatalf("invalid name error code = %v, want invalid", coreerrors.AsCode(err))
	}
}

func TestParseCom0comList(t *testing.T) {
	output := `
CNCA0 PortName=COM#,RealPortName=COM12
CNCB0 PortName=COM#,RealPortName=COM13
CNCA1 PortName=COM20
CNCB1 PortName=CNCB1
`
	pairs := parseCom0comList(output)
	if len(pairs) != 2 {
		t.Fatalf("pairs len = %d, want 2: %#v", len(pairs), pairs)
	}
	if pairs[0].number != "0" || pairs[0].portA != "COM12" || pairs[0].portB != "COM13" {
		t.Fatalf("pair[0] = %#v", pairs[0])
	}
	if pairs[1].number != "1" || pairs[1].portA != "COM20" || pairs[1].portB != "CNCB1" {
		t.Fatalf("pair[1] = %#v", pairs[1])
	}
}

func TestFindNewCom0comPairHelpersOnlyMatchNewPairs(t *testing.T) {
	before := []com0comListPair{{number: "0", portA: "COM10", portB: "COM11"}}
	pairs := []com0comListPair{
		{number: "0", portA: "COM10", portB: "COM11"},
		{number: "1", portA: "COM12", portB: "COM13"},
		{number: "2", portA: "COM14", portB: "COM15"},
	}
	seen := com0comPairNumbers(before)
	matches := make([]string, 0)
	wantA := windowsPortSpec{exact: "COM12"}
	wantB := windowsPortSpec{exact: "COM13"}
	for _, pair := range pairs {
		if seen[pair.number] {
			continue
		}
		if windowsPortMatches(pair.portA, wantA) && windowsPortMatches(pair.portB, wantB) {
			matches = append(matches, pair.number)
		}
	}
	if len(matches) != 1 || matches[0] != "1" {
		t.Fatalf("matches = %#v, want only new matching pair 1", matches)
	}
}
