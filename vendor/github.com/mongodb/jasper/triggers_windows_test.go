package jasper

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mongodStartupTime = 15 * time.Second

func TestMongodShutdownEventTrigger(t *testing.T) {
	for procName, makeProc := range map[string]ProcessConstructor{
		"Basic":    newBasicProcess,
		"Blocking": newBlockingProcess,
	} {
		t.Run(procName, func(t *testing.T) {
			for testName, testParams := range map[string]struct {
				signal               syscall.Signal
				useMongod            bool
				expectCleanTerminate bool
			}{
				"WithSIGTERMAndMongod":       {signal: syscall.SIGTERM, useMongod: true, expectCleanTerminate: true},
				"WithNonSIGTERMAndMongod":    {signal: syscall.SIGHUP, useMongod: true, expectCleanTerminate: false},
				"WithSIGTERMAndNonMongod":    {signal: syscall.SIGTERM, useMongod: false, expectCleanTerminate: false},
				"WithNonSIGTERMAndNonMongod": {signal: syscall.SIGHUP, useMongod: false, expectCleanTerminate: false},
			} {
				t.Run(testName, func(t *testing.T) {
					if testing.Short() {
						t.Skip("skipping mongod shutdown tests in short mode")
					}

					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					var opts CreateOptions
					if testParams.useMongod {
						dir, mongodPath := downloadMongoDB(t)
						defer os.RemoveAll(dir)

						optslist, dbPaths, err := setupMongods(1, mongodPath)
						require.NoError(t, err)
						defer removeDBPaths(dbPaths)
						require.Equal(t, 1, len(optslist))

						opts = optslist[0]
						opts.Output.Loggers = []Logger{Logger{Type: LogDefault, Options: LogOptions{Format: LogFormatPlain}}}
					} else {
						opts = yesCreateOpts(0)
					}

					proc, err := makeProc(ctx, &opts)
					require.NoError(t, err)

					if testParams.useMongod {
						// Give mongod time to start up and create the termination event.
						time.Sleep(mongodStartupTime)
					}

					trigger := makeMongodShutdownSignalTrigger()
					terminated := trigger(proc.Info(ctx), testParams.signal)
					if testParams.expectCleanTerminate {
						assert.True(t, terminated)
					} else {
						assert.False(t, terminated)
					}

					if testParams.expectCleanTerminate {
						exitCode, err := proc.Wait(ctx)
						assert.NoError(t, err)
						assert.Zero(t, exitCode)
						assert.False(t, proc.Running(ctx))
					} else {
						assert.True(t, proc.Running(ctx))
						assert.NoError(t, proc.Signal(ctx, syscall.SIGKILL))
					}
				})
			}
		})
	}
}

func TestCleanTerminationSignalTrigger(t *testing.T) {
	for procName, makeProc := range map[string]ProcessConstructor{
		"Basic":    newBasicProcess,
		"Blocking": newBlockingProcess,
	} {
		t.Run(procName, func(t *testing.T) {
			for testName, testCase := range map[string]func(context.Context, *CreateOptions, ProcessConstructor){
				"CleanTerminationRunsForSIGTERM": func(ctx context.Context, opts *CreateOptions, makep ProcessConstructor) {
					proc, err := makep(ctx, opts)
					require.NoError(t, err)
					trigger := makeCleanTerminationSignalTrigger()
					assert.True(t, trigger(proc.Info(ctx), syscall.SIGTERM))

					exitCode, err := proc.Wait(ctx)
					assert.NoError(t, err)
					assert.Zero(t, exitCode)
					assert.False(t, proc.Running(ctx))

					// Subsequent executions of trigger should fail.
					assert.False(t, trigger(proc.Info(ctx), syscall.SIGTERM))
				},
				"CleanTerminationIgnoresNonSIGTERM": func(ctx context.Context, opts *CreateOptions, makep ProcessConstructor) {
					proc, err := makep(ctx, opts)
					require.NoError(t, err)
					trigger := makeCleanTerminationSignalTrigger()
					assert.False(t, trigger(proc.Info(ctx), syscall.SIGHUP))

					assert.True(t, proc.Running(ctx))

					assert.NoError(t, proc.Signal(ctx, syscall.SIGKILL))
				},
				"CleanTerminationFailsForExitedProcess": func(ctx context.Context, opts *CreateOptions, makep ProcessConstructor) {
					opts = trueCreateOpts()
					proc, err := makep(ctx, opts)
					require.NoError(t, err)

					exitCode, err := proc.Wait(ctx)
					assert.NoError(t, err)
					assert.Zero(t, exitCode)

					trigger := makeCleanTerminationSignalTrigger()
					assert.False(t, trigger(proc.Info(ctx), syscall.SIGTERM))
				},
			} {
				t.Run(testName, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					opts := yesCreateOpts(0)
					testCase(ctx, &opts, makeProc)
				})
			}
		})
	}
}
