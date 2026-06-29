package module

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// fakeModule is a controllable Module for registry tests. It records the
// order of lifecycle calls and can be configured to fail at any stage.
type fakeModule struct {
	id       string
	deps     []string
	frontend FrontendContribution
	services []application.Service

	mu       sync.Mutex
	calls    []string
	initErr  error
	startErr error
	stopErr  error
	disposed bool
}

func (m *fakeModule) ID() string { return m.id }
func (m *fakeModule) Manifest() Manifest {
	return Manifest{ID: m.id, Dependencies: m.deps, Frontend: m.frontend}
}
func (m *fakeModule) Init(ctx context.Context, d Deps) error {
	m.record("init:" + m.id)
	return m.initErr
}
func (m *fakeModule) Services() []any { return nil }
func (m *fakeModule) ServicesWrapped() []application.Service {
	return m.services
}
func (m *fakeModule) Start(ctx context.Context) error {
	m.record("start:" + m.id)
	return m.startErr
}
func (m *fakeModule) Stop(ctx context.Context) error {
	m.record("stop:" + m.id)
	return m.stopErr
}
func (m *fakeModule) Dispose() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disposed = true
	m.calls = append(m.calls, "dispose:"+m.id)
}

func (m *fakeModule) record(s string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, s)
}
func (m *fakeModule) snapshot() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.calls))
	copy(out, m.calls)
	return out
}

func newFake(id string, deps ...string) *fakeModule {
	return &fakeModule{id: id, deps: deps}
}

func mustRegister(t *testing.T, r *ModuleRegistry, modules ...Module) {
	t.Helper()
	for _, m := range modules {
		if err := r.Register(m); err != nil {
			t.Fatalf("Register(%q) failed: %v", m.ID(), err)
		}
	}
}

type registryWrappedService struct {
	name string
}

func TestRegisterRejectsDuplicateID(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register(newFake("a")); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}
	if err := r.Register(newFake("a")); err == nil {
		t.Fatalf("duplicate Register must error")
	}
}

func TestRegisterRejectsEmptyID(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register(newFake("")); err == nil {
		t.Fatalf("empty ID must error")
	}
}

func TestTopologicalOrderRespectsDependencies(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	// a depends on b, b depends on c => order must be c, b, a.
	if err := r.Register(newFake("a", "b")); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(newFake("b", "c")); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(newFake("c")); err != nil {
		t.Fatal(err)
	}

	r.mu.RLock()
	order, err := r.orderLocked()
	r.mu.RUnlock()
	if err != nil {
		t.Fatalf("order failed: %v", err)
	}
	if len(order) != 3 || order[0] != "c" || order[1] != "b" || order[2] != "a" {
		t.Fatalf("order = %v, want [c b a]", order)
	}
}

func TestTopologicalOrderDetectsCycle(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	_ = r.Register(newFake("a", "b"))
	_ = r.Register(newFake("b", "a"))
	r.mu.RLock()
	_, err := r.orderLocked()
	r.mu.RUnlock()
	if err == nil {
		t.Fatalf("cyclic dependency must error")
	}
}

func TestTopologicalOrderDetectsMissingDependency(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	_ = r.Register(newFake("a", "ghost"))
	r.mu.RLock()
	_, err := r.orderLocked()
	r.mu.RUnlock()
	if err == nil {
		t.Fatalf("missing dependency must error")
	}
}

func TestInitAllRollsBackOnFailure(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	c := newFake("c")
	b := newFake("b", "c")
	b.initErr = errors.New("boom")
	a := newFake("a", "b")
	mustRegister(t, r, a, b, c)

	err := r.InitAll(context.Background(), Deps{})
	if err == nil {
		t.Fatalf("InitAll must surface the init error")
	}
	// c was init'd, then b failed: c should have been disposed (rollback).
	if !c.disposed {
		t.Fatalf("rolled-back module c must be disposed")
	}
}

func TestStartAllRollsBackOnFailure(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	c := newFake("c")
	b := newFake("b", "c")
	b.startErr = errors.New("boom")
	a := newFake("a", "b")
	mustRegister(t, r, a, b, c)

	if err := r.InitAll(context.Background(), Deps{}); err != nil {
		t.Fatalf("InitAll failed: %v", err)
	}
	if err := r.StartAll(context.Background()); err == nil {
		t.Fatalf("StartAll must surface the start error")
	}
	// c started, then b failed: c must have been stopped (and both disposed).
	if len(c.snapshot()) == 0 || lastCall(c) != "dispose:c" {
		t.Fatalf("rolled-back module c must end with dispose, calls=%v", c.snapshot())
	}
}

func TestStopAllContinuesOnPartialFailure(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	c := newFake("c")
	b := newFake("b", "c")
	b.stopErr = errors.New("boom")
	a := newFake("a", "b")
	mustRegister(t, r, a, b, c)

	_ = r.InitAll(context.Background(), Deps{})
	_ = r.StartAll(context.Background())

	err := r.StopAll(context.Background())
	if err == nil {
		t.Fatalf("StopAll must report the partial failure")
	}
	// Even though b.Stop errored, all modules must have been stopped.
	for _, m := range []*fakeModule{a, b, c} {
		found := false
		for _, call := range m.snapshot() {
			if call == "stop:"+m.id {
				found = true
			}
		}
		if !found {
			t.Fatalf("module %s was not stopped, calls=%v", m.id, m.snapshot())
		}
	}
}

func TestAllServicesWrappedAggregatesServicesInDependencyOrder(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	baseInstance := &registryWrappedService{name: "base"}
	dependentInstance := &registryWrappedService{name: "dependent"}
	dependent := newFake("dependent", "base")
	dependent.services = []application.Service{application.NewService(dependentInstance)}
	base := newFake("base")
	base.services = []application.Service{application.NewService(baseInstance)}
	_ = r.Register(dependent)
	_ = r.Register(base)

	services := r.AllServicesWrapped()
	if len(services) != 2 {
		t.Fatalf("services length = %d, want 2", len(services))
	}
	if services[0].Instance() != baseInstance || services[1].Instance() != dependentInstance {
		t.Fatalf("services aggregated in wrong order: %#v then %#v", services[0].Instance(), services[1].Instance())
	}
}

func TestFrontendContributionsIncludeModuleIDAndManifestInDependencyOrder(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	dependent := newFake("dependent", "base")
	dependent.frontend = FrontendContribution{
		ActivityTitle: "Dependent",
		Views:         []FrontendView{{ID: "dependent.view", Title: "Dependent View", Component: "DependentView"}},
	}
	base := newFake("base")
	base.frontend = FrontendContribution{
		ActivityTitle: "Base",
		Views:         []FrontendView{{ID: "base.view", Title: "Base View", Component: "BaseView"}},
	}
	_ = r.Register(dependent)
	_ = r.Register(base)

	contributions := r.FrontendContributions()
	if len(contributions) != 2 {
		t.Fatalf("contributions length = %d, want 2", len(contributions))
	}
	if contributions[0].ModuleID != "base" || contributions[0].Manifest.Frontend.Views[0].ID != "base.view" {
		t.Fatalf("first contribution = %+v, want base manifest", contributions[0])
	}
	if contributions[1].ModuleID != "dependent" || contributions[1].Manifest.Frontend.Views[0].ID != "dependent.view" {
		t.Fatalf("second contribution = %+v, want dependent manifest", contributions[1])
	}
}

func lastCall(m *fakeModule) string {
	s := m.snapshot()
	if len(s) == 0 {
		return ""
	}
	return s[len(s)-1]
}

func TestRegisterConcurrentIsSafe(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = r.Register(newFake(fmt.Sprintf("m%d", n)))
		}(i)
	}
	wg.Wait()
	// All 50 distinct IDs should be registered.
	r.mu.RLock()
	count := len(r.modules)
	r.mu.RUnlock()
	if count != 50 {
		t.Fatalf("registered %d modules, want 50", count)
	}
}
