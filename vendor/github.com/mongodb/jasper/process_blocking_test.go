package jasper

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestBlockingProcess(t *testing.T) {
	t.Parallel()
	// we run the suite multiple times given that implementation
	// is heavily threaded, there are timing concerns that require
	// multiple executions.
	for _, attempt := range []string{"First", "Second", "Third", "Fourth", "Fifth"} {
		t.Run(attempt, func(t *testing.T) {
			t.Parallel()
			for name, testCase := range map[string]func(context.Context, *testing.T, *blockingProcess){
				"VerifyTestCaseConfiguration": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					assert.NotNil(t, proc)
					assert.NotNil(t, ctx)
					assert.NotZero(t, proc.ID())
					assert.False(t, proc.Complete(ctx))
					assert.NotNil(t, makeDefaultTrigger(ctx, nil, &proc.opts, "foo"))
				},
				"InfoIDPopulatedInBasicCase": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					proc.info = &ProcessInfo{
						ID: proc.ID(),
					}

					info := proc.Info(ctx)
					assert.Equal(t, info.ID, proc.ID())
				},
				"InfoRetrunsZeroValueForCanceledCase": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					signal := make(chan struct{})
					go func() {
						cctx, cancel := context.WithCancel(ctx)
						cancel()

						assert.Zero(t, proc.Info(cctx))
						close(signal)
					}()

					go func() {
						<-proc.ops
					}()

					gracefulCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
					defer cancel()

					select {
					case <-signal:
					case <-gracefulCtx.Done():
						t.Error("reached timeout")
					}
				},
				"SignalErrorsForCancledContext": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					signal := make(chan struct{})
					go func() {

						cctx, cancel := context.WithCancel(ctx)
						cancel()

						assert.Error(t, proc.Signal(cctx, syscall.SIGTERM))
						close(signal)
					}()

					go func() {
						<-proc.ops
					}()

					gracefulCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
					defer cancel()

					select {
					case <-signal:
					case <-gracefulCtx.Done():
						t.Error("reached timeout")
					}
				},
				"TestRegisterTriggerAfterComplete": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					proc.info = &ProcessInfo{}

					assert.True(t, proc.Complete(ctx))
					assert.Error(t, proc.RegisterTrigger(ctx, nil))
					assert.Error(t, proc.RegisterTrigger(ctx, makeDefaultTrigger(ctx, nil, &proc.opts, "foo")))
					assert.Len(t, proc.triggers, 0)
				},
				"TestRegisterPopulatedTrigger": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					assert.False(t, proc.Complete(ctx))
					assert.Error(t, proc.RegisterTrigger(ctx, nil))
					assert.NoError(t, proc.RegisterTrigger(ctx, makeDefaultTrigger(ctx, nil, &proc.opts, "foo")))
					assert.Len(t, proc.triggers, 1)
				},
				"RunningIsFalseWhenCompleteIsSatisfied": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					proc.info = &ProcessInfo{}
					assert.True(t, proc.Complete(ctx))
					assert.False(t, proc.Running(ctx))
				},
				"RunningIsFalseWithEmptyPid": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					signal := make(chan struct{})
					go func() {
						assert.False(t, proc.Running(ctx))
						close(signal)
					}()

					op := <-proc.ops

					op(&exec.Cmd{
						Process: &os.Process{},
					})
					<-signal
				},
				"RunningIsFalseWithNilCmd": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					signal := make(chan struct{})
					go func() {
						assert.False(t, proc.Running(ctx))
						close(signal)
					}()

					op := <-proc.ops
					op(nil)

					<-signal
				},
				"RunningIsTrueWithValidPid": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					signal := make(chan struct{})
					go func() {
						assert.True(t, proc.Running(ctx))
						close(signal)
					}()

					op := <-proc.ops
					op(&exec.Cmd{
						Process: &os.Process{Pid: 42},
					})

					<-signal
				},
				"RunningIsFalseWithCancledContext": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					proc.ops <- func(_ *exec.Cmd) {}
					cctx, cancel := context.WithCancel(ctx)
					cancel()
					assert.False(t, proc.Running(cctx))
				},
				"SignalIsErrorAfterComplete": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					proc.info = &ProcessInfo{}
					assert.True(t, proc.Complete(ctx))

					assert.Error(t, proc.Signal(ctx, syscall.SIGTERM))
				},
				"SignalNilProcessIsError": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					signal := make(chan struct{})
					go func() {
						assert.Nil(t, proc.info)
						assert.Error(t, proc.Signal(ctx, syscall.SIGTERM))
						close(signal)
					}()

					op := <-proc.ops
					op(nil)

					<-signal
				},
				"SignalCancledProcessIsError": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					cctx, cancel := context.WithCancel(ctx)
					cancel()

					assert.Error(t, proc.Signal(cctx, syscall.SIGTERM))
				},
				"SignalErrorsInvalidProcess": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					signal := make(chan struct{})
					go func() {
						assert.Nil(t, proc.info)
						assert.Error(t, proc.Signal(ctx, syscall.SIGTERM))
						close(signal)
					}()

					op := <-proc.ops
					op(&exec.Cmd{
						Process: &os.Process{Pid: -42},
					})

					<-signal
				},
				"WaitSomeBeforeCanceling": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					proc.opts.Args = []string{"sleep", "1"}
					cctx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
					defer cancel()

					cmd, err := proc.opts.Resolve(ctx)
					assert.NoError(t, err)
					assert.NoError(t, cmd.Start())
					startAt := time.Now()
					go proc.reactor(ctx, cmd)
					err = proc.Wait(cctx)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "operation canceled")
					assert.True(t, time.Since(startAt) >= 500*time.Millisecond)
				},
				"WaitShouldReturnNilForSuccessfulCommandsWithoutIDs": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					proc.opts.Args = []string{"sleep", "10"}
					proc.ops = make(chan func(*exec.Cmd))

					cmd, err := proc.opts.Resolve(ctx)
					assert.NoError(t, err)
					assert.NoError(t, cmd.Start())
					signal := make(chan struct{})
					go func() {
						// this is the crucial
						// assertion of this tests
						assert.NoError(t, proc.Wait(ctx))
						close(signal)
					}()

					go func() {
						for {
							select {
							case op := <-proc.ops:
								proc.setInfo(ProcessInfo{
									Successful: true,
								})
								if op != nil {
									op(cmd)
								}
							}
						}
					}()
					<-signal
				},
				"WaitShouldReturnNilForSuccessfulCommands": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					proc.opts.Args = []string{"sleep", "10"}
					proc.ops = make(chan func(*exec.Cmd))

					cmd, err := proc.opts.Resolve(ctx)
					assert.NoError(t, err)
					assert.NoError(t, cmd.Start())
					signal := make(chan struct{})
					go func() {
						// this is the crucial
						// assertion of this tests
						assert.NoError(t, proc.Wait(ctx))
						close(signal)
					}()

					go func() {
						for {
							select {
							case op := <-proc.ops:
								proc.setInfo(ProcessInfo{
									ID:         "foo",
									Successful: true,
								})
								if op != nil {
									op(cmd)
								}
							}
						}
					}()
					<-signal
				},
				"WaitShouldReturnErrorForFailedCommands": func(ctx context.Context, t *testing.T, proc *blockingProcess) {
					proc.opts.Args = []string{"sleep", "10"}
					proc.ops = make(chan func(*exec.Cmd))

					cmd, err := proc.opts.Resolve(ctx)
					assert.NoError(t, err)
					assert.NoError(t, cmd.Start())
					signal := make(chan struct{})
					go func() {
						// this is the crucial assertion
						// of this tests.
						assert.Error(t, proc.Wait(ctx))
						close(signal)
					}()

					go func() {
						for {
							select {
							case op := <-proc.ops:
								proc.setInfo(ProcessInfo{
									ID:         "foo",
									Successful: false,
								})
								if op != nil {
									op(cmd)
								}
							}
						}
					}()
					<-signal
				},
				// "": func(ctx context.Context, t *testing.T, proc *blockingProcess) {},
			} {
				t.Run(name, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					proc := &blockingProcess{
						id:   uuid.Must(uuid.NewV4()).String(),
						ops:  make(chan func(*exec.Cmd), 1),
						opts: CreateOptions{},
					}

					testCase(ctx, t, proc)

					close(proc.ops)
				})
			}
		})
	}
}
