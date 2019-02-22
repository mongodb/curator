// +build windows

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
	for procName, makeProc := range map[string]processConstructor{
		"Basic":    newBasicProcess,
		"Blocking": newBlockingProcess,
	} {
		t.Run(procName, func(t *testing.T) {
			for testName, testParams := range map[string]struct {
				signal               syscall.Signal
				useMongod            bool
				expectCleanTerminate bool
			}{
				"WithSIGTERMAndMongod":           {signal: syscall.SIGTERM, useMongod: true, expectCleanTerminate: true},
				"WithSIGKILLAndMongod":           {signal: syscall.SIGKILL, useMongod: true, expectCleanTerminate: true},
				"WithNonTerminationAndMongod":    {signal: syscall.SIGHUP, useMongod: true, expectCleanTerminate: false},
				"WithSIGTERMAndNonMongod":        {signal: syscall.SIGTERM, useMongod: false, expectCleanTerminate: false},
				"WithSIGKILLAndNonMongod":        {signal: syscall.SIGKILL, useMongod: false, expectCleanTerminate: false},
				"WithNonTerminationAndNonMongod": {signal: syscall.SIGHUP, useMongod: false, expectCleanTerminate: false},
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
					trigger(proc.Info(ctx), testParams.signal)

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
