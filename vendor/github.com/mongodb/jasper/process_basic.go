package jasper

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

type basicProcess struct {
	id       string
	hostname string
	opts     CreateOptions
	cmd      *exec.Cmd
	tags     map[string]struct{}
	triggers ProcessTriggerSequence
}

func newBasicProcess(ctx context.Context, opts *CreateOptions) (Process, error) {
	id := uuid.Must(uuid.NewV4()).String()
	opts.AddEnvVar(EnvironID, id)

	cmd, err := opts.Resolve(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "problem building command from options")
	}

	p := &basicProcess{
		id:       id,
		opts:     *opts,
		cmd:      cmd,
		tags:     make(map[string]struct{}),
		triggers: ProcessTriggerSequence{},
	}
	p.hostname, _ = os.Hostname()

	for _, t := range opts.Tags {
		p.Tag(t)
	}

	p.RegisterTrigger(ctx, makeOptionsCloseTrigger())

	err = cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, "problem creating command")
	}

	p.opts.started = true
	opts.started = true

	return p, nil
}

func (p *basicProcess) ID() string { return p.id }
func (p *basicProcess) Info(ctx context.Context) ProcessInfo {
	if ctx.Err() != nil {
		return ProcessInfo{}
	}

	info := ProcessInfo{
		ID:        p.id,
		Options:   p.opts,
		Host:      p.hostname,
		Complete:  p.Complete(ctx),
		IsRunning: p.Running(ctx),
	}

	if info.Complete {
		info.Successful = p.cmd.ProcessState.Success()
		info.ExitCode = p.cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
		info.PID = -1
	}

	if info.IsRunning {
		info.PID = p.cmd.Process.Pid
		info.ExitCode = -1
	}

	return info
}

func (p *basicProcess) Complete(ctx context.Context) bool {
	if p.cmd == nil {
		return false
	}

	if p.cmd.ProcessState != nil && p.cmd.ProcessState.Exited() {
		return true
	}

	if p.cmd.Process == nil {
		return false
	}

	return p.cmd.Process.Pid == -1
}

func (p *basicProcess) Running(ctx context.Context) bool {
	// if we haven't created the command or it hasn't started than
	// it isn't running
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}

	if p.cmd.Process.Pid <= 0 {
		return false
	}

	// if we have a viable pid then it's (probably) running
	return true
}

func (p *basicProcess) Signal(ctx context.Context, sig syscall.Signal) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return errors.New("cannot signal nil process")
	}
	return errors.Wrapf(p.cmd.Process.Signal(sig), "problem sending signal '%s' to '%s'", sig, p.id)
}

func (p *basicProcess) Wait(ctx context.Context) error {
	if p.Complete(ctx) {
		return nil
	}

	if p.cmd == nil {
		return errors.New("process is not defined")
	}

	sig := make(chan error)
	go func() {
		defer close(sig)

		select {
		case sig <- p.cmd.Wait():
			p.triggers.Run(p.Info(ctx))
		case <-ctx.Done():
			select {
			case sig <- ctx.Err():
			default:
			}
		}

		return
	}()

	select {
	case <-ctx.Done():
		return errors.New("context canceled while waiting for process to exit")
	case err := <-sig:
		return errors.WithStack(err)
	}
}

func (p *basicProcess) RegisterTrigger(ctx context.Context, trigger ProcessTrigger) error {
	if p.Complete(ctx) {
		return errors.New("cannot register trigger after process exits")
	}

	if trigger == nil {
		return errors.New("cannot register nil trigger")
	}

	p.triggers = append(p.triggers, trigger)

	return nil
}

func (p *basicProcess) Tag(t string) {
	_, ok := p.tags[t]
	if ok {
		return
	}

	p.tags[t] = struct{}{}
	p.opts.Tags = append(p.opts.Tags, t)
}

func (p *basicProcess) ResetTags() {
	p.tags = make(map[string]struct{})
	p.opts.Tags = []string{}
}

func (p *basicProcess) GetTags() []string {
	out := []string{}
	for t := range p.tags {
		out = append(out, t)
	}
	return out
}
