package jasper

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

func NewLocalManager() Manager {
	return &localProcessManager{
		manager: &basicProcessManager{
			procs:    map[string]Process{},
			blocking: false,
		},
	}
}

func NewLocalManagerBlockingProcesses() Manager {
	return &localProcessManager{
		manager: &basicProcessManager{
			procs:    map[string]Process{},
			blocking: true,
		},
	}
}

type localProcessManager struct {
	mu      sync.RWMutex
	manager *basicProcessManager
}

func (m *localProcessManager) Create(ctx context.Context, opts *CreateOptions) (Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.manager.skipDefaultTrigger = true
	proc, err := m.manager.Create(ctx, opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	proc.RegisterTrigger(ctx, makeDefaultTrigger(ctx, m, opts, proc.ID()))

	proc = &localProcess{proc: proc}
	m.manager.procs[proc.ID()] = proc

	return proc, nil
}

func (m *localProcessManager) Register(ctx context.Context, proc Process) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return errors.WithStack(m.manager.Register(ctx, proc))
}

func (m *localProcessManager) List(ctx context.Context, f Filter) ([]Process, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	procs, err := m.manager.List(ctx, f)
	return procs, errors.WithStack(err)
}

func (m *localProcessManager) Get(ctx context.Context, id string) (Process, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	proc, err := m.manager.Get(ctx, id)
	return proc, errors.WithStack(err)
}

func (m *localProcessManager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return errors.WithStack(m.manager.Close(ctx))
}

func (m *localProcessManager) Group(ctx context.Context, name string) ([]Process, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	procs, err := m.manager.Group(ctx, name)
	return procs, errors.WithStack(err)
}
