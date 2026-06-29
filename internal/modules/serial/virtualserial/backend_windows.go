//go:build windows

package virtualserial

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/littepointR/portweave/internal/core/errors"
)

const com0comSetupEnv = "PORTWEAVE_COM0COM_SETUPC"

type com0comBackend struct{}

// DefaultBackend returns the platform virtual serial backend.
func DefaultBackend() Backend {
	return com0comBackend{}
}

func (com0comBackend) Name() string { return "com0com" }

func (b com0comBackend) Status(ctx context.Context) BackendStatus {
	setupc, err := b.setupcPath()
	if err != nil {
		return BackendStatus{
			Name:          b.Name(),
			Available:     false,
			Message:       "com0com setupc.exe was not found",
			Reason:        err.Error(),
			RequiresAdmin: true,
		}
	}
	if _, err := runSetupc(ctx, setupc, "list"); err != nil {
		return BackendStatus{
			Name:          b.Name(),
			Available:     false,
			Message:       "com0com setupc.exe is present but the backend is not ready",
			Reason:        err.Error(),
			RequiresAdmin: true,
		}
	}
	return BackendStatus{
		Name:          b.Name(),
		Available:     true,
		Message:       "com0com virtual serial backend is available",
		Reason:        setupc,
		RequiresAdmin: true,
	}
}

func (b com0comBackend) CreatePair(ctx context.Context, pairID, port1Name, port2Name string) (*VirtualPair, error) {
	setupc, err := b.setupcPath()
	if err != nil {
		return nil, errors.Wrap(errors.CodePlatform, "com0com setupc.exe not found", err)
	}
	port1, err := windowsVisiblePortSpec(port1Name)
	if err != nil {
		return nil, err
	}
	port2, err := windowsVisiblePortSpec(port2Name)
	if err != nil {
		return nil, err
	}

	before, err := listCom0comPairs(ctx, setupc)
	if err != nil {
		return nil, err
	}
	args := []string{
		"install",
		port1.arg,
		port2.arg,
	}
	if _, err := runSetupc(ctx, setupc, args...); err != nil {
		return nil, classifyCom0comError("create com0com pair", err)
	}

	pair, err := findNewCom0comPair(ctx, setupc, before, port1, port2)
	if err != nil {
		cleanupMatchingNewCom0comPairs(setupc, before, func(pair com0comListPair) bool {
			return port1.exact != "" && port2.exact != "" &&
				windowsPortMatches(pair.portA, port1) && windowsPortMatches(pair.portB, port2)
		})
		return nil, err
	}
	return newVirtualPair(pairID, strings.ToUpper(pair.portA), strings.ToUpper(pair.portB), com0comPairHandle{
		setupc: setupc,
		number: pair.number,
	}), nil
}

func (b com0comBackend) CreatePort(ctx context.Context, portID, publicName string) (*VirtualPair, error) {
	setupc, err := b.setupcPath()
	if err != nil {
		return nil, errors.Wrap(errors.CodePlatform, "com0com setupc.exe not found", err)
	}
	publicPort, err := windowsVisiblePortSpec(publicName)
	if err != nil {
		return nil, err
	}

	before, err := listCom0comPairs(ctx, setupc)
	if err != nil {
		return nil, err
	}
	args := []string{
		"install",
		publicPort.arg,
		"-",
	}
	if _, err := runSetupc(ctx, setupc, args...); err != nil {
		return nil, classifyCom0comError("create com0com port", err)
	}

	pair, publicOnA, err := findNewCom0comPairWithPublic(ctx, setupc, before, publicPort)
	if err != nil {
		cleanupMatchingNewCom0comPairs(setupc, before, func(pair com0comListPair) bool {
			return publicPort.exact != "" &&
				((windowsPortMatches(pair.portA, publicPort) && pair.portB != "") ||
					(windowsPortMatches(pair.portB, publicPort) && pair.portA != ""))
		})
		return nil, err
	}
	canonicalA, canonicalB := strings.ToUpper(pair.portA), strings.ToUpper(pair.portB)
	if !publicOnA {
		canonicalA, canonicalB = canonicalB, canonicalA
	}
	return newVirtualPair(portID, canonicalA, canonicalB, com0comPairHandle{
		setupc: setupc,
		number: pair.number,
	}), nil
}

func (com0comBackend) setupcPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv(com0comSetupEnv)); override != "" {
		return existingFile(override)
	}

	if exePath, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "com0com", "setupc.exe")
		if path, err := existingFile(candidate); err == nil {
			return path, nil
		}
	}

	if path, err := exec.LookPath("setupc.exe"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("setupc"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("checked %s, <app>\\com0com\\setupc.exe, and PATH", com0comSetupEnv)
}

func existingFile(path string) (string, error) {
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", clean)
	}
	return clean, nil
}

func normalizeWindowsPortName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errors.New(errors.CodeInvalid, "port name must not be empty")
	}
	upper := strings.ToUpper(trimmed)
	if strings.HasPrefix(upper, `\\.\`) {
		upper = strings.TrimPrefix(upper, `\\.\`)
	}
	if !isWindowsCOMName(upper) {
		return "", errors.New(errors.CodeInvalid, "Windows virtual serial ports must use COM<number> names")
	}
	return upper, nil
}

type windowsPortSpec struct {
	arg   string
	exact string
}

func windowsVisiblePortSpec(name string) (windowsPortSpec, error) {
	exact, err := normalizeWindowsPortName(name)
	if err == nil {
		return windowsPortSpec{arg: "PortName=" + exact, exact: exact}, nil
	}
	if coreCode := errors.AsCode(err); coreCode != errors.CodeInvalid {
		return windowsPortSpec{}, err
	}
	if strings.TrimSpace(name) == "" {
		return windowsPortSpec{}, err
	}
	return windowsPortSpec{arg: "PortName=COM#"}, nil
}

func isWindowsCOMName(name string) bool {
	matched, _ := regexp.MatchString(`^COM[1-9][0-9]*$`, name)
	return matched
}

func runSetupc(ctx context.Context, setupc string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmdCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, setupc, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	text := strings.TrimSpace(output.String())
	if cmdCtx.Err() != nil {
		return text, cmdCtx.Err()
	}
	if err != nil {
		if text == "" {
			return "", err
		}
		return text, fmt.Errorf("%w: %s", err, text)
	}
	return text, nil
}

func classifyCom0comError(action string, err error) error {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "access is denied"), strings.Contains(message, "elevat"), strings.Contains(message, "administrator"):
		return errors.Wrap(errors.CodePlatform, action+" requires administrator privileges", err)
	case strings.Contains(message, "already"), strings.Contains(message, "exist"), strings.Contains(message, "in use"):
		return errors.Wrap(errors.CodeConflict, action+" failed because a COM name is already in use", err)
	default:
		return errors.Wrap(errors.CodeIO, action, err)
	}
}

type com0comListPair struct {
	number string
	portA  string
	portB  string
}

func listCom0comPairs(ctx context.Context, setupc string) ([]com0comListPair, error) {
	output, err := runSetupc(ctx, setupc, "list")
	if err != nil {
		return nil, classifyCom0comError("list com0com pairs", err)
	}
	return parseCom0comList(output), nil
}

func findNewCom0comPair(ctx context.Context, setupc string, before []com0comListPair, wantA, wantB windowsPortSpec) (com0comListPair, error) {
	pairs, err := listCom0comPairs(ctx, setupc)
	if err != nil {
		return com0comListPair{}, err
	}
	seen := com0comPairNumbers(before)
	for _, pair := range pairs {
		if seen[pair.number] {
			continue
		}
		if windowsPortMatches(pair.portA, wantA) && windowsPortMatches(pair.portB, wantB) {
			return pair, nil
		}
	}
	return com0comListPair{}, errors.New(errors.CodeIO, "created com0com pair was not found in setupc list output")
}

func findNewCom0comPairWithPublic(ctx context.Context, setupc string, before []com0comListPair, publicPort windowsPortSpec) (com0comListPair, bool, error) {
	pairs, err := listCom0comPairs(ctx, setupc)
	if err != nil {
		return com0comListPair{}, false, err
	}
	seen := com0comPairNumbers(before)
	for _, pair := range pairs {
		if seen[pair.number] {
			continue
		}
		if windowsPortMatches(pair.portA, publicPort) && pair.portB != "" {
			return pair, true, nil
		}
		if windowsPortMatches(pair.portB, publicPort) && pair.portA != "" {
			return pair, false, nil
		}
	}
	return com0comListPair{}, false, errors.New(errors.CodeIO, "created com0com port was not found in setupc list output")
}

func windowsPortMatches(actual string, spec windowsPortSpec) bool {
	actual = strings.ToUpper(strings.TrimSpace(actual))
	if spec.exact != "" {
		return strings.EqualFold(actual, spec.exact)
	}
	return isWindowsCOMName(actual)
}

func com0comPairNumbers(pairs []com0comListPair) map[string]bool {
	result := make(map[string]bool, len(pairs))
	for _, pair := range pairs {
		result[pair.number] = true
	}
	return result
}

func cleanupMatchingNewCom0comPairs(setupc string, before []com0comListPair, matches func(com0comListPair) bool) {
	if matches == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pairs, err := listCom0comPairs(ctx, setupc)
	if err != nil {
		return
	}
	seen := com0comPairNumbers(before)
	for _, pair := range pairs {
		if seen[pair.number] || !matches(pair) {
			continue
		}
		_ = removeCom0comPair(ctx, setupc, pair.number)
	}
}

func parseCom0comList(output string) []com0comListPair {
	var pairs []com0comListPair
	lines := strings.Split(output, "\n")
	pending := map[string]string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		device := strings.ToUpper(fields[0])
		if !strings.HasPrefix(device, "CNCA") && !strings.HasPrefix(device, "CNCB") {
			continue
		}
		number := device[4:]
		portName := device
		realPortName := ""
		for _, field := range fields[1:] {
			for _, token := range strings.Split(field, ",") {
				key, value, ok := strings.Cut(token, "=")
				if !ok || value == "" {
					continue
				}
				switch {
				case strings.EqualFold(key, "RealPortName"):
					realPortName = strings.ToUpper(value)
				case strings.EqualFold(key, "PortName") && value != "-":
					portName = strings.ToUpper(value)
				}
			}
		}
		if realPortName != "" {
			portName = realPortName
		}
		if strings.HasPrefix(device, "CNCA") {
			pending["A"+number] = portName
		} else {
			pending["B"+number] = portName
		}
		if a, okA := pending["A"+number]; okA {
			if b, okB := pending["B"+number]; okB {
				pairs = append(pairs, com0comListPair{number: number, portA: a, portB: b})
			}
		}
	}
	return pairs
}

type com0comPairHandle struct {
	setupc string
	number string
}

func (h com0comPairHandle) Stop() error {
	return removeCom0comPair(context.Background(), h.setupc, h.number)
}

func removeCom0comPair(ctx context.Context, setupc, number string) error {
	if setupc == "" || number == "" {
		return nil
	}
	if _, err := runSetupc(ctx, setupc, "remove", number); err != nil {
		return classifyCom0comError("remove com0com pair", err)
	}
	return nil
}
