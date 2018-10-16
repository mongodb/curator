package jasper

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type basicProcessManager struct {
	procs              map[string]Process
	blocking           bool
	skipDefaultTrigger bool
}

func (m *basicProcessManager) Create(ctx context.Context, opts *CreateOptions) (Process, error) {
	var (
		proc Process
		err  error
	)

	if m.blocking {
		proc, err = newBlockingProcess(ctx, opts)
	} else {
		proc, err = newBasicProcess(ctx, opts)
	}

	if err != nil {
		return nil, errors.Wrap(err, "problem constructing local process")
	}

	// TODO this will race because it runs later
	if !m.skipDefaultTrigger {
		proc.RegisterTrigger(ctx, makeDefaultTrigger(ctx, m, opts, proc.ID()))
	}

	m.procs[proc.ID()] = proc

	return proc, nil
}

func (m *basicProcessManager) Register(ctx context.Context, proc Process) error {
	if ctx.Err() != nil {
		return errors.New("context canceled")
	}

	if proc == nil {
		return errors.New("process is not defined")
	}

	id := proc.ID()
	if id == "" {
		return errors.New("process is malformed")
	}

	_, ok := m.procs[id]
	if ok {
		return errors.New("cannot register process that exists")
	}

	m.procs[id] = proc
	return nil
}

func (m *basicProcessManager) List(ctx context.Context, f Filter) ([]Process, error) {
	out := []Process{}

	for _, proc := range m.procs {
		if ctx.Err() != nil {
			return nil, errors.New("operation canceled")
		}

		info := proc.Info(ctx)
		switch {
		case f == Running:
			if info.IsRunning {
				out = append(out, proc)
			}
			continue
		case f == Successful:
			if info.Successful {
				out = append(out, proc)
			}
			continue
		case f == Failed:
			if info.Complete && !info.Successful {
				out = append(out, proc)
			}
			continue
		case f == All:
			out = append(out, proc)
			continue
		}
	}

	if len(out) == 0 {
		return nil, errors.New("no processes")
	}

	return out, nil
}

func (m *basicProcessManager) Get(ctx context.Context, id string) (Process, error) {
	proc, ok := m.procs[id]
	if !ok {
		return nil, errors.Errorf("process '%s' does not exist", id)
	}

	return proc, nil
}

func (m *basicProcessManager) Close(ctx context.Context) error {
	if len(m.procs) == 0 {
		return nil
	}
	procs, err := m.List(ctx, Running)
	if err != nil {
		return errors.WithStack(err)
	}

	termCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := TerminateAll(termCtx, procs); err != nil {
		killCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		return errors.WithStack(KillAll(killCtx, procs))
	}

	return nil
}

func (m *basicProcessManager) Group(ctx context.Context, name string) ([]Process, error) {
	out := []Process{}
	for _, proc := range m.procs {
		if ctx.Err() != nil {
			return nil, errors.New("request canceled")
		}

		if sliceContains(proc.GetTags(), name) {
			out = append(out, proc)
		}
	}

	if len(out) == 0 {
		return nil, errors.Errorf("no jobs tagged '%s'", name)
	}

	return out, nil
}
