package jasper

import (
	"context"
	"sync"
	"syscall"

	"github.com/pkg/errors"
)

type localProcess struct {
	proc  Process
	mutex sync.RWMutex
}

func (p *localProcess) ID() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.proc.ID()
}

func (p *localProcess) Info(ctx context.Context) ProcessInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.proc.Info(ctx)
}

func (p *localProcess) Running(ctx context.Context) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.proc.Running(ctx)
}

func (p *localProcess) Complete(ctx context.Context) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.proc.Complete(ctx)
}

func (p *localProcess) Signal(ctx context.Context, sig syscall.Signal) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return errors.WithStack(p.proc.Signal(ctx, sig))
}

func (p *localProcess) Tag(t string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.proc.Tag(t)
}

func (p *localProcess) ResetTags() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.proc.ResetTags()
}

func (p *localProcess) GetTags() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.proc.GetTags()
}

func (p *localProcess) RegisterTrigger(ctx context.Context, trigger ProcessTrigger) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return errors.WithStack(p.proc.RegisterTrigger(ctx, trigger))
}

func (p *localProcess) Wait(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return errors.WithStack(p.proc.Wait(ctx))
}
