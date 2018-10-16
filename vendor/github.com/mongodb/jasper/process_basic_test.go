package jasper

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestBasicProcess(t *testing.T) {
	for name, testCase := range map[string]func(context.Context, *testing.T, *basicProcess){
		"VerifyTestCaseConfiguration": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			assert.Nil(t, proc.cmd)
			assert.NotNil(t, proc)
			assert.NotNil(t, ctx)
			assert.NotZero(t, proc.ID())
			assert.NotNil(t, makeDefaultTrigger(ctx, nil, &proc.opts, "foo"))
		},
		"InfoIDPopulatedInBasicCase": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			info := proc.Info(ctx)
			assert.Equal(t, info.ID, proc.ID())
		},
		"ContextCanceledReturnsEmptyValue": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			cctx, cancel := context.WithCancel(ctx)
			cancel()
			info := proc.Info(cctx)
			assert.Zero(t, info)
		},
		"CompleteIsFalseWithoutCmdPopulated": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			assert.False(t, proc.Complete(ctx))
		},
		"RunningIsFalseWithoutCmdPopulated": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			assert.False(t, proc.Running(ctx))
		},
		"RunningProcessAppearsRunning": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			proc.opts.Args = []string{"sleep", "100"}
			cmd, err := proc.opts.Resolve(ctx)
			assert.NoError(t, err)
			proc.cmd = cmd
			assert.NoError(t, proc.cmd.Start())
			assert.True(t, proc.Running(ctx))
		},
		"UnstartedCommandsAppearRunning": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			proc.cmd = &exec.Cmd{
				Process:      &os.Process{},
				ProcessState: &os.ProcessState{},
			}

			assert.NotNil(t, proc.cmd)
			assert.NotNil(t, proc.cmd.Process)
			assert.False(t, proc.Running(ctx))
		},
		"TestRegisterTriggerAfterComplete": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			proc.cmd = &exec.Cmd{
				Process:      &os.Process{Pid: 42},
				ProcessState: &os.ProcessState{},
			}

			assert.True(t, proc.Complete(ctx))
			assert.Error(t, proc.RegisterTrigger(ctx, nil))
			assert.Error(t, proc.RegisterTrigger(ctx, makeDefaultTrigger(ctx, nil, &proc.opts, "foo")))
			assert.Len(t, proc.triggers, 0)
		},
		"TestRegisterPopulatedTrigger": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			proc.cmd = &exec.Cmd{
				Process: &os.Process{Pid: 42},
			}

			assert.False(t, proc.Complete(ctx))
			assert.Error(t, proc.RegisterTrigger(ctx, nil))
			assert.NoError(t, proc.RegisterTrigger(ctx, makeDefaultTrigger(ctx, nil, &proc.opts, "foo")))
			assert.Len(t, proc.triggers, 1)
		},
		"SignalShouldErrorInNilcase": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			assert.Error(t, proc.Signal(ctx, syscall.SIGTERM))
		},
		"SignalShouldErrorInStoppedCase": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			proc.cmd = &exec.Cmd{
				Process: &os.Process{Pid: -1},
			}

			assert.Error(t, proc.Signal(ctx, syscall.SIGTERM))
		},
		"WaitDoesNotErrorForCompleteProcesses": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			proc.cmd = &exec.Cmd{
				Process: &os.Process{Pid: -1},
			}

			assert.True(t, proc.Complete(ctx))
			assert.NoError(t, proc.Wait(ctx))
		},
		"WaitReturnsWithUndefinedProcess": func(ctx context.Context, t *testing.T, proc *basicProcess) {
			assert.Error(t, proc.Wait(ctx))
		},
		// "": func(ctx context.Context, t *testing.T, proc *basicProcess) {},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			testCase(ctx, t, &basicProcess{
				id:   uuid.Must(uuid.NewV4()).String(),
				opts: CreateOptions{},
			})
		})
	}
}
