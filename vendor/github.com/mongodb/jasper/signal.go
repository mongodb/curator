package jasper

import (
	"context"
	"runtime"
	"syscall"

	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// Terminate sends a SIGTERM signal to the given process under the given
// context. This does not guarantee that the process will actually die. This
// function does not Wait() on the given process upon sending the signal. On
// Windows, this function is a no-op.
func Terminate(ctx context.Context, p Process) error {
	// TODO: MAKE-570: Update signal.go functions with Windows-specific behaviors.
	if runtime.GOOS == "windows" {
		return nil
	}
	return errors.WithStack(p.Signal(ctx, syscall.SIGTERM))
}

// Kill sends a SIGKILL signal to the given process under the given context.
// This guarantees that the process will die. This function does not Wait() on
// the given process upon sending the signal. On Windows, this function is a
// no-op.
func Kill(ctx context.Context, p Process) error {
	// TODO: MAKE-570: Update signal.go functions with Windows-specific behaviors.
	if runtime.GOOS == "windows" {
		return nil
	}
	return errors.WithStack(p.Signal(ctx, syscall.SIGKILL))
}

// TerminateAll sends a SIGTERM signal to each of the given processes under the
// given context. This does not guarantee that each process will actually die.
// This function calls Wait() on each process after sending them SIGTERM
// signals. On Windows, this function does not send a SIGTERM signal, but will
// Wait() on each process until it exits. Use Terminate() in a loop if you do
// not wish to potentially hang on Wait().
func TerminateAll(ctx context.Context, procs []Process) error {
	catcher := grip.NewBasicCatcher()

	for _, proc := range procs {
		if proc.Running(ctx) {
			catcher.Add(Terminate(ctx, proc))
		}
	}

	for _, proc := range procs {
		proc.Wait(ctx)
	}

	return catcher.Resolve()
}

// KillAll sends a SIGKILL signal to each of the given processes under the
// given context. This guarantees that each process will actually die.  This
// function calls Wait() on each process after sending them SIGKILL signals. On
// Windows, this function does not send a SIGKILL signal, but will Wait() on
// each process until it exits. Use Kill() in a loop if you do not wish to
// potentially hang on Wait().
func KillAll(ctx context.Context, procs []Process) error {
	catcher := grip.NewBasicCatcher()

	for _, proc := range procs {
		if proc.Running(ctx) {
			catcher.Add(Kill(ctx, proc))
		}
	}

	for _, proc := range procs {
		proc.Wait(ctx)
	}

	return catcher.Resolve()
}
