package module

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	mterrors "github.com/littepointR/portweave/internal/core/errors"
)

// ModuleRegistry manages module registration and the ordered Init/Start/Stop
// lifecycle with rollback on failure. All methods are safe for concurrent
// use.
type ModuleRegistry struct {
	mu      sync.RWMutex
	modules map[string]Module
	logger  *slog.Logger
}

// NewRegistry constructs an empty registry.
func NewRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]Module),
		logger:  slog.Default(),
	}
}

// Register adds a module. Duplicate or empty IDs are rejected.
func (r *ModuleRegistry) Register(m Module) error {
	id := m.ID()
	if id == "" {
		return mterrors.New(mterrors.CodeInvalid, "module ID must not be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.modules[id]; exists {
		return mterrors.New(mterrors.CodeConflict, fmt.Sprintf("module already registered: %s", id))
	}
	r.modules[id] = m
	return nil
}

// InitAll calls Init on every module in topological order, injecting deps.
// On the first failure it disposes already-initialised modules plus the
// failing module itself (reverse order) and returns the wrapped error.
func (r *ModuleRegistry) InitAll(ctx context.Context, deps Deps) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, err := r.orderLocked()
	if err != nil {
		return err
	}

	initialised := make([]Module, 0, len(order))
	for _, id := range order {
		m := r.modules[id]
		if err := m.Init(ctx, deps); err != nil {
			// The failing module may have allocated partial resources; ask
			// it to dispose too, then roll back the already-initialised ones.
			m.Dispose()
			for i := len(initialised) - 1; i >= 0; i-- {
				initialised[i].Dispose()
			}
			return mterrors.Wrap(mterrors.CodeModuleInit, fmt.Sprintf("init module %s", id), err)
		}
		initialised = append(initialised, m)
	}
	return nil
}

// StartAll calls Start on every module in topological order. On the first
// failure it stops and disposes already-started modules (reverse order) and
// returns the wrapped error.
func (r *ModuleRegistry) StartAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, err := r.orderLocked()
	if err != nil {
		return err
	}
	started := make([]Module, 0, len(order))
	for _, id := range order {
		m := r.modules[id]
		if err := m.Start(ctx); err != nil {
			for i := len(started) - 1; i >= 0; i-- {
				_ = started[i].Stop(ctx)
				started[i].Dispose()
			}
			return mterrors.Wrap(mterrors.CodeModuleStart, fmt.Sprintf("start module %s", id), err)
		}
		started = append(started, m)
	}
	return nil
}

// StopAll calls Stop on every module in reverse topological order. A failure
// in one module does not stop the others; the first error is returned.
func (r *ModuleRegistry) StopAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, err := r.orderLocked()
	if err != nil {
		return err
	}
	var firstErr error
	for i := len(order) - 1; i >= 0; i-- {
		m := r.modules[order[i]]
		if err := m.Stop(ctx); err != nil && firstErr == nil {
			firstErr = mterrors.Wrap(mterrors.CodeModuleStop, fmt.Sprintf("stop module %s", order[i]), err)
		}
	}
	return firstErr
}

// DisposeAll calls Dispose on every module in reverse topological order.
// Errors are ignored (Dispose is idempotent and must not fail). A topological
// failure is logged but does not panic.
func (r *ModuleRegistry) DisposeAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, err := r.orderLocked()
	if err != nil {
		r.logger.Error("module registry: cannot compute order for dispose", slog.Any("err", err))
		return
	}
	for i := len(order) - 1; i >= 0; i-- {
		r.modules[order[i]].Dispose()
	}
}

// AllServicesWrapped aggregates every module's wrapped services in
// topological order, ready for application.Options.Services. Returns nil if
// the dependency graph is invalid (the error is logged).
func (r *ModuleRegistry) AllServicesWrapped() []application.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, err := r.orderLocked()
	if err != nil {
		r.logger.Error("module registry: cannot compute order for services", slog.Any("err", err))
		return nil
	}
	out := make([]application.Service, 0)
	for _, id := range order {
		out = append(out, r.modules[id].ServicesWrapped()...)
	}
	return out
}

// FrontendContributions returns each module's manifest plus its ID, for the
// shell to emit to the frontend so it can register views. Returns nil if the
// dependency graph is invalid (the error is logged).
func (r *ModuleRegistry) FrontendContributions() []ModuleContribution {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, err := r.orderLocked()
	if err != nil {
		r.logger.Error("module registry: cannot compute order for contributions", slog.Any("err", err))
		return nil
	}
	out := make([]ModuleContribution, 0, len(order))
	for _, id := range order {
		m := r.modules[id]
		out = append(out, ModuleContribution{
			ModuleID: id,
			Manifest: m.Manifest(),
		})
	}
	return out
}

// ModuleContribution pairs a module ID with its manifest for the frontend.
type ModuleContribution struct {
	ModuleID string
	Manifest Manifest
}

// order returns module IDs in topological order, detecting cycles and
// missing dependencies. Caller must hold at least a read lock.
func (r *ModuleRegistry) orderLocked() ([]string, error) {
	// Kahn's algorithm.
	indegree := make(map[string]int, len(r.modules))
	dependents := make(map[string][]string)
	for id, m := range r.modules {
		indegree[id] = 0
		for _, dep := range m.Manifest().Dependencies {
			if _, ok := r.modules[dep]; !ok {
				return nil, mterrors.New(mterrors.CodeNotFound, fmt.Sprintf("module %s depends on missing module %s", id, dep))
			}
			indegree[id]++
			dependents[dep] = append(dependents[dep], id)
		}
	}

	var queue []string
	for id, d := range indegree {
		if d == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, dep := range dependents[id] {
			indegree[dep]--
			if indegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(order) != len(r.modules) {
		return nil, mterrors.New(mterrors.CodeInvalid, "circular module dependency detected")
	}
	return order, nil
}
