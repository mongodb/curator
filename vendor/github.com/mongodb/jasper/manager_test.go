package jasper

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagerInterface(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for mname, factory := range map[string]func(ctx context.Context, t *testing.T) Manager{
		"Basic/NoLock/BlockingProcs": func(ctx context.Context, t *testing.T) Manager {
			return &basicProcessManager{
				procs: map[string]Process{},
			}
		},
		"Basic/Lock/BlockingProcs": func(ctx context.Context, t *testing.T) Manager {
			return NewLocalManager()
		},
		"Basic/Lock/SelfClearing/BlockingProcs": func(ctx context.Context, t *testing.T) Manager {
			return NewSelfClearingProcessManager(10)
		},
		"REST": func(ctx context.Context, t *testing.T) Manager {
			srv, port := makeAndStartService(ctx, httpClient)
			require.NotNil(t, srv)

			return &restClient{
				prefix: fmt.Sprintf("http://localhost:%d/jasper/v1", port),
				client: httpClient,
			}
		},
	} {
		t.Run(mname, func(t *testing.T) {
			for name, test := range map[string]func(context.Context, *testing.T, Manager){
				"ValidateFixture": func(ctx context.Context, t *testing.T, manager Manager) {
					assert.NotNil(t, ctx)
					assert.NotNil(t, manager)
				},
				"ListErrorsWhenEmpty": func(ctx context.Context, t *testing.T, manager Manager) {
					all, err := manager.List(ctx, All)
					assert.Error(t, err)
					assert.Len(t, all, 0)
					assert.Contains(t, err.Error(), "no processes")
				},
				"CreateSimpleProcess": func(ctx context.Context, t *testing.T, manager Manager) {
					if mname == "REST" {
						t.Skip("test case not compatible with rest interfaces")
					}

					opts := trueCreateOpts()
					proc, err := manager.Create(ctx, opts)
					assert.NoError(t, err)
					assert.NotNil(t, proc)
					assert.True(t, opts.started)
				},
				"CreateProcessFails": func(ctx context.Context, t *testing.T, manager Manager) {
					proc, err := manager.Create(ctx, &CreateOptions{})
					assert.Error(t, err)
					assert.Nil(t, proc)
				},
				"ListAllOperations": func(ctx context.Context, t *testing.T, manager Manager) {
					t.Skip("this often deadlocks")
					created, err := createProcs(ctx, trueCreateOpts(), manager, 10)
					assert.NoError(t, err)
					assert.Len(t, created, 10)
					output, err := manager.List(ctx, All)
					assert.NoError(t, err)
					assert.Len(t, output, 10)
				},
				"ListAllReturnsErrorWithCancledContext": func(ctx context.Context, t *testing.T, manager Manager) {
					cctx, cancel := context.WithCancel(ctx)
					created, err := createProcs(ctx, trueCreateOpts(), manager, 10)
					assert.NoError(t, err)
					assert.Len(t, created, 10)
					cancel()
					output, err := manager.List(cctx, All)
					assert.Error(t, err)
					assert.Nil(t, output)
				},
				"LongRunningOperationsAreListedAsRunning": func(ctx context.Context, t *testing.T, manager Manager) {
					procs, err := createProcs(ctx, sleepCreateOpts(1), manager, 10)
					assert.NoError(t, err)
					assert.Len(t, procs, 10)

					procs, err = manager.List(ctx, Running)
					assert.NoError(t, err)
					assert.Len(t, procs, 10)

					procs, err = manager.List(ctx, Successful)
					assert.Error(t, err)
					assert.Len(t, procs, 0)
				},
				"ListReturnsOneSuccessfulCommand": func(ctx context.Context, t *testing.T, manager Manager) {
					proc, err := manager.Create(ctx, trueCreateOpts())
					require.NoError(t, err)

					_, err = proc.Wait(ctx)
					assert.NoError(t, err)

					listOut, err := manager.List(ctx, Successful)
					assert.NoError(t, err)

					if assert.Len(t, listOut, 1) {
						assert.Equal(t, listOut[0].ID(), proc.ID())
					}
				},
				"ListReturnsOneFailedCommand": func(ctx context.Context, t *testing.T, manager Manager) {
					proc, err := manager.Create(ctx, falseCreateOpts())
					require.NoError(t, err)
					_, err = proc.Wait(ctx)
					assert.Error(t, err)

					listOut, err := manager.List(ctx, Failed)
					assert.NoError(t, err)

					if assert.Len(t, listOut, 1) {
						assert.Equal(t, listOut[0].ID(), proc.ID())
					}
				},
				"GetMethodErrorsWithNoResponse": func(ctx context.Context, t *testing.T, manager Manager) {
					proc, err := manager.Get(ctx, "foo")
					assert.Error(t, err)
					assert.Nil(t, proc)
				},
				"GetMethodReturnsMatchingDoc": func(ctx context.Context, t *testing.T, manager Manager) {
					proc, err := manager.Create(ctx, trueCreateOpts())
					require.NoError(t, err)

					ret, err := manager.Get(ctx, proc.ID())
					assert.NoError(t, err)
					assert.Equal(t, ret.ID(), proc.ID())
				},
				"GroupErrorsWithoutResults": func(ctx context.Context, t *testing.T, manager Manager) {
					procs, err := manager.Group(ctx, "foo")
					assert.Error(t, err)
					assert.Len(t, procs, 0)
					assert.Contains(t, err.Error(), "no jobs")
				},
				"GroupErrorsForCanceledContexts": func(ctx context.Context, t *testing.T, manager Manager) {
					_, err := manager.Create(ctx, trueCreateOpts())
					assert.NoError(t, err)

					cctx, cancel := context.WithCancel(ctx)
					cancel()
					procs, err := manager.Group(cctx, "foo")
					assert.Error(t, err)
					assert.Len(t, procs, 0)
					assert.Contains(t, err.Error(), "canceled")
				},
				"GroupPropgatesMatching": func(ctx context.Context, t *testing.T, manager Manager) {
					proc, err := manager.Create(ctx, trueCreateOpts())
					require.NoError(t, err)

					proc.Tag("foo")

					procs, err := manager.Group(ctx, "foo")
					require.NoError(t, err)
					require.Len(t, procs, 1)
					assert.Equal(t, procs[0].ID(), proc.ID())
				},
				"CloseEmptyManagerNoops": func(ctx context.Context, t *testing.T, manager Manager) {
					assert.NoError(t, manager.Close(ctx))
				},
				"CloseErrorsWithCanceledContext": func(ctx context.Context, t *testing.T, manager Manager) {
					_, err := createProcs(ctx, sleepCreateOpts(100), manager, 10)
					assert.NoError(t, err)

					cctx, cancel := context.WithCancel(ctx)
					cancel()

					err = manager.Close(cctx)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "canceled")
				},
				"CloseErrorsWithTerminatedProcesses": func(ctx context.Context, t *testing.T, manager Manager) {
					procs, err := createProcs(ctx, trueCreateOpts(), manager, 10)
					for _, p := range procs {
						_, err := p.Wait(ctx)
						assert.NoError(t, err)
					}

					assert.NoError(t, err)
					assert.Error(t, manager.Close(ctx))
				},
				"ClosersWithoutTriggersTerminatesProcesses": func(ctx context.Context, t *testing.T, manager Manager) {
					if runtime.GOOS == "windows" {
						t.Skip("the sleep tests don't block correctly on windows")
					}

					_, err := createProcs(ctx, sleepCreateOpts(100), manager, 10)
					assert.NoError(t, err)
					assert.NoError(t, manager.Close(ctx))
				},
				"CloseExcutesClosersForProcesses": func(ctx context.Context, t *testing.T, manager Manager) {
					if mname == "REST" {
						t.Skip("not supported on rest interfaces")
					}

					if runtime.GOOS == "windows" {
						t.Skip("the sleep tests don't block correctly on windows")
					}

					opts := sleepCreateOpts(20)
					counter := 0
					opts.closers = append(opts.closers, func() {
						counter++
					})
					closersDone := make(chan bool)
					opts.closers = append(opts.closers, func() { closersDone <- true })

					_, err := manager.Create(ctx, opts)
					assert.NoError(t, err)

					assert.Equal(t, counter, 0)
					assert.NoError(t, manager.Close(ctx))
					select {
					case <-ctx.Done():
						assert.Fail(t, "process took too long to run closers")
					case <-closersDone:
						assert.Equal(t, 1, counter)
					}
				},
				"RegisterProcessErrorsForNilProcess": func(ctx context.Context, t *testing.T, manager Manager) {
					if mname == "REST" {
						t.Skip("not supported on rest interfaces")
					}

					err := manager.Register(ctx, nil)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "not defined")
				},
				"RegisterProcessErrorsForCancledContext": func(ctx context.Context, t *testing.T, manager Manager) {
					if mname == "REST" {
						t.Skip("not supported on rest interfaces")
					}

					cctx, cancel := context.WithCancel(ctx)
					cancel()
					proc, err := newBlockingProcess(ctx, trueCreateOpts())
					assert.NoError(t, err)
					err = manager.Register(cctx, proc)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "canceled")
				},
				"RegisterProcessErrorsWhenMissingID": func(ctx context.Context, t *testing.T, manager Manager) {
					if mname == "REST" {
						t.Skip("not supported on rest interfaces")
					}

					proc := &blockingProcess{}
					assert.Equal(t, proc.ID(), "")
					err := manager.Register(ctx, proc)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "malformed")
				},
				"RegisterProcessModifiesManagerState": func(ctx context.Context, t *testing.T, manager Manager) {
					if mname == "REST" {
						t.Skip("not supported on rest interfaces")
					}

					proc, err := newBlockingProcess(ctx, trueCreateOpts())
					require.NoError(t, err)
					err = manager.Register(ctx, proc)
					assert.NoError(t, err)

					procs, err := manager.List(ctx, All)
					assert.NoError(t, err)
					require.True(t, len(procs) >= 1)

					assert.Equal(t, procs[0].ID(), proc.ID())
				},
				"RegisterProcessErrorsForDuplicateProcess": func(ctx context.Context, t *testing.T, manager Manager) {
					if mname == "REST" {
						t.Skip("not supported on rest interfaces")
					}

					proc, err := newBlockingProcess(ctx, trueCreateOpts())
					assert.NoError(t, err)
					assert.NotEmpty(t, proc)
					err = manager.Register(ctx, proc)
					assert.NoError(t, err)
					err = manager.Register(ctx, proc)
					assert.Error(t, err)
				},
				"ManagerCallsOptionsCloseByDefault": func(ctx context.Context, t *testing.T, manager Manager) {
					if mname == "REST" {
						t.Skip("cannot register trigger on rest interfaces")
					}

					opts := &CreateOptions{}
					opts.Args = []string{"echo", "foobar"}
					count := 0
					opts.closers = append(opts.closers, func() { count++ })
					closersDone := make(chan bool)
					opts.closers = append(opts.closers, func() { closersDone <- true })
					proc, err := manager.Create(ctx, opts)
					assert.NoError(t, err)
					_, err = proc.Wait(ctx)
					assert.NoError(t, err)
					select {
					case <-ctx.Done():
						assert.Fail(t, "process took too long to run closers")
					case <-closersDone:
						assert.Equal(t, 1, count)
					}
				},
				"ClearCausesDeletionOfProcesses": func(ctx context.Context, t *testing.T, manager Manager) {
					opts := trueCreateOpts()
					proc, err := manager.Create(ctx, opts)
					require.NoError(t, err)
					sameProc, err := manager.Get(ctx, proc.ID())
					require.Equal(t, proc.ID(), sameProc.ID())
					_, err = proc.Wait(ctx)
					require.NoError(t, err)
					manager.Clear(ctx)
					nilProc, err := manager.Get(ctx, proc.ID())
					assert.Nil(t, nilProc)
				},
				"ClearIsANoOpForActiveProcesses": func(ctx context.Context, t *testing.T, manager Manager) {
					opts := sleepCreateOpts(20)
					proc, err := manager.Create(ctx, opts)
					require.NoError(t, err)
					manager.Clear(ctx)
					sameProc, err := manager.Get(ctx, proc.ID())
					assert.Equal(t, proc.ID(), sameProc.ID())
					require.NoError(t, Terminate(ctx, proc)) // Clean up
				},
				"ClearSelectivelyDeletesOnlyDeadProcesses": func(ctx context.Context, t *testing.T, manager Manager) {
					trueOpts := trueCreateOpts()
					lsProc, err := manager.Create(ctx, trueOpts)
					require.NoError(t, err)

					sleepOpts := sleepCreateOpts(20)
					sleepProc, err := manager.Create(ctx, sleepOpts)
					require.NoError(t, err)

					_, err = lsProc.Wait(ctx)
					require.NoError(t, err)

					manager.Clear(ctx)

					sameSleepProc, err := manager.Get(ctx, sleepProc.ID())
					require.NoError(t, err)
					assert.Equal(t, sleepProc.ID(), sameSleepProc.ID())

					nilProc, err := manager.Get(ctx, lsProc.ID())
					assert.Nil(t, nilProc)
					require.NoError(t, Terminate(ctx, sleepProc)) // Clean up
				},
				// "": func(ctx context.Context, t *testing.T, manager Manager) {},
			} {
				t.Run(name, func(t *testing.T) {
					tctx, cancel := context.WithTimeout(ctx, managerTestTimeout)
					defer cancel()
					test(tctx, t, factory(tctx, t))
				})
			}
		})
	}
}
