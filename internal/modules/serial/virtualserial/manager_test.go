package virtualserial

import (
	"context"
	"errors"
	"testing"

	coreerrors "github.com/littepointR/portweave/internal/core/errors"
)

type fakeBackend struct {
	status    BackendStatus
	pairErr   error
	portErr   error
	stopErr   error
	created   []string
	stopCount int
}

func (f *fakeBackend) Name() string { return "fake" }

func (f *fakeBackend) Status(context.Context) BackendStatus {
	if f.status.Name == "" {
		return BackendStatus{Name: "fake", Available: true, Message: "ok"}
	}
	return f.status
}

func (f *fakeBackend) CreatePair(_ context.Context, pairID, port1Name, port2Name string) (*VirtualPair, error) {
	if f.pairErr != nil {
		return nil, f.pairErr
	}
	f.created = append(f.created, "pair:"+pairID+":"+port1Name+":"+port2Name)
	return newVirtualPair(pairID, port1Name, port2Name, fakeHandle{backend: f}), nil
}

func (f *fakeBackend) CreatePort(_ context.Context, portID, publicName string) (*VirtualPair, error) {
	if f.portErr != nil {
		return nil, f.portErr
	}
	f.created = append(f.created, "port:"+portID+":"+publicName)
	return newVirtualPair(portID, publicName, publicName+"-hidden", fakeHandle{backend: f}), nil
}

type fakeHandle struct {
	backend *fakeBackend
}

func (h fakeHandle) Stop() error {
	h.backend.stopCount++
	return h.backend.stopErr
}

func TestManagerUsesInjectedBackendForPairsAndPorts(t *testing.T) {
	backend := &fakeBackend{}
	mgr := NewManager(WithBackend(backend))

	pair, err := mgr.CreatePair(context.Background(), "pair-1", "A", "B")
	if err != nil {
		t.Fatalf("CreatePair: %v", err)
	}
	if pair.Port1 != "A" || pair.Port2 != "B" || pair.IsUserFacingPort() {
		t.Fatalf("pair = %+v, userFacing=%v", pair, pair.IsUserFacingPort())
	}

	port, err := mgr.CreatePort(context.Background(), "port-1", "COM10")
	if err != nil {
		t.Fatalf("CreatePort: %v", err)
	}
	if port.Port1 != "COM10" || port.Port2 != "COM10-hidden" || !port.IsUserFacingPort() {
		t.Fatalf("port = %+v, userFacing=%v", port, port.IsUserFacingPort())
	}
	if endpoint := mgr.EndpointFor("COM10"); endpoint != "COM10-hidden" {
		t.Fatalf("EndpointFor(COM10) = %q, want hidden peer", endpoint)
	}
	if endpoint := mgr.EndpointFor("A"); endpoint != "A" {
		t.Fatalf("EndpointFor(A) = %q, want public pair endpoint", endpoint)
	}
	if got, want := len(backend.created), 2; got != want {
		t.Fatalf("created calls = %d, want %d", got, want)
	}

	mgr.Cleanup()
	if backend.stopCount != 2 {
		t.Fatalf("stopCount = %d, want 2", backend.stopCount)
	}
}

func TestManagerBackendStatusAndErrors(t *testing.T) {
	backend := &fakeBackend{
		status:  BackendStatus{Name: "fake", Available: false, Message: "missing", Reason: "not installed"},
		pairErr: errors.New("pair denied"),
		portErr: coreerrors.New(coreerrors.CodePlatform, "unavailable"),
	}
	mgr := NewManager(WithBackend(backend))

	status := mgr.BackendStatus(context.Background())
	if status.Available || status.Name != "fake" || status.Reason != "not installed" {
		t.Fatalf("status = %+v", status)
	}

	if _, err := mgr.CreatePair(context.Background(), "pair-1", "A", "B"); err == nil {
		t.Fatalf("CreatePair must return backend error")
	}
	if _, err := mgr.CreatePort(context.Background(), "port-1", "COM10"); coreerrors.AsCode(err) != coreerrors.CodePlatform {
		t.Fatalf("CreatePort error code = %v, want platform: %v", coreerrors.AsCode(err), err)
	}
}

func TestManagerDeletePairPreservesStateWhenBackendCleanupFails(t *testing.T) {
	backend := &fakeBackend{stopErr: errors.New("remove denied")}
	mgr := NewManager(WithBackend(backend))
	if _, err := mgr.CreatePort(context.Background(), "port-1", "COM10"); err != nil {
		t.Fatalf("CreatePort: %v", err)
	}

	if err := mgr.DeletePair("port-1"); err == nil {
		t.Fatalf("DeletePair must return backend cleanup error")
	}
	if got := len(mgr.ListPairs()); got != 1 {
		t.Fatalf("ListPairs after failed delete = %d, want 1", got)
	}

	backend.stopErr = nil
	if err := mgr.DeletePair("port-1"); err != nil {
		t.Fatalf("DeletePair retry: %v", err)
	}
	if got := len(mgr.ListPairs()); got != 0 {
		t.Fatalf("ListPairs after retry = %d, want 0", got)
	}
}
