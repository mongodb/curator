package remote

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mongodb/jasper/scripting"
	"github.com/mongodb/jasper/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScripting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpClient := testutil.GetHTTPClient()
	defer testutil.PutHTTPClient(httpClient)

	for managerName, makeManager := range remoteManagerTestCases(httpClient) {
		t.Run(managerName, func(t *testing.T) {
			for _, test := range []clientTestCase{
				{
					Name: "ScriptingSetupSucceeds",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()
						harness := createTestScriptingHarness(ctx, t, client, tmpDir)
						assert.NoError(t, harness.Setup(ctx))
					},
				},
				{
					Name: "ScriptingCleanupSucceeds",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()
						harness := createTestScriptingHarness(ctx, t, client, tmpDir)
						assert.NoError(t, harness.Cleanup(ctx))
					},
				},
				{
					Name: "ScriptingRunSucceeds",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()
						harness := createTestScriptingHarness(ctx, t, client, tmpDir)

						require.NoError(t, err)
						tmpFile := filepath.Join(tmpDir, "fake_script.go")
						require.NoError(t, ioutil.WriteFile(tmpFile, []byte(testutil.GolangMainSuccess()), 0755))
						assert.NoError(t, harness.Run(ctx, []string{tmpFile}))
					},
				},
				{
					Name: "ScriptingRunErrors",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()
						harness := createTestScriptingHarness(ctx, t, client, tmpDir)

						tmpFile := filepath.Join(tmpDir, "fake_script.go")
						require.NoError(t, ioutil.WriteFile(tmpFile, []byte(testutil.GolangMainFail()), 0755))
						assert.Error(t, harness.Run(ctx, []string{tmpFile}))
					},
				},
				{
					Name: "ScriptingRunScriptSucceeds",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()
						harness := createTestScriptingHarness(ctx, t, client, tmpDir)
						assert.NoError(t, harness.RunScript(ctx, testutil.GolangMainSuccess()))
					},
				},
				{
					Name: "ScriptingRunScriptErrors",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()

						harness := createTestScriptingHarness(ctx, t, client, tmpDir)
						require.Error(t, harness.RunScript(ctx, testutil.GolangMainFail()))
					},
				},
				{
					Name: "ScriptingBuildSucceeds",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()
						harness := createTestScriptingHarness(ctx, t, client, tmpDir)

						tmpFile := filepath.Join(tmpDir, "fake_script.go")
						require.NoError(t, ioutil.WriteFile(tmpFile, []byte(testutil.GolangMainSuccess()), 0755))
						buildFile := filepath.Join(tmpDir, "fake_script")
						_, err = harness.Build(ctx, tmpDir, []string{
							"-o",
							buildFile,
							tmpFile,
						})
						require.NoError(t, err)
						_, err = os.Stat(buildFile)
						require.NoError(t, err)
					},
				},
				{
					Name: "ScriptingBuildErrors",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()
						harness := createTestScriptingHarness(ctx, t, client, tmpDir)

						tmpFile := filepath.Join(tmpDir, "fake_script.go")
						require.NoError(t, ioutil.WriteFile(tmpFile, []byte(`package main; func main() { "bad syntax" }`), 0755))
						buildFile := filepath.Join(tmpDir, "fake_script")
						_, err = harness.Build(ctx, tmpDir, []string{
							"-o",
							buildFile,
							tmpFile,
						})
						require.Error(t, err)
						_, err = os.Stat(buildFile)
						assert.Error(t, err)
					},
				},
				{
					Name: "ScriptingTestSucceeds",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()
						harness := createTestScriptingHarness(ctx, t, client, tmpDir)

						tmpFile := filepath.Join(tmpDir, "fake_script_test.go")
						require.NoError(t, ioutil.WriteFile(tmpFile, []byte(testutil.GolangTestSuccess()), 0755))
						results, err := harness.Test(ctx, tmpDir, scripting.TestOptions{Name: "dummy"})
						require.NoError(t, err)
						require.Len(t, results, 1)
						assert.Equal(t, scripting.TestOutcomeSuccess, results[0].Outcome)
					},
				},
				{
					Name: "ScriptingTestErrors",
					Case: func(ctx context.Context, t *testing.T, client Manager) {
						tmpDir, err := ioutil.TempDir(testutil.BuildDirectory(), "scripting_tests")
						require.NoError(t, err)
						defer func() {
							assert.NoError(t, os.RemoveAll(tmpDir))
						}()
						harness := createTestScriptingHarness(ctx, t, client, tmpDir)

						tmpFile := filepath.Join(tmpDir, "fake_script_test.go")
						require.NoError(t, ioutil.WriteFile(tmpFile, []byte(testutil.GolangTestFail()), 0755))
						results, err := harness.Test(ctx, tmpDir, scripting.TestOptions{Name: "dummy"})
						assert.Error(t, err)
						require.Len(t, results, 1)
						assert.Equal(t, scripting.TestOutcomeFailure, results[0].Outcome)
					},
				},
			} {
				t.Run(test.Name, func(t *testing.T) {
					tctx, cancel := context.WithTimeout(ctx, testutil.RPCTestTimeout)
					defer cancel()
					client := makeManager(tctx, t)
					defer func() {
						assert.NoError(t, client.CloseConnection())
					}()
					test.Case(tctx, t, client)
				})

			}
		})
	}
}

func createTestScriptingHarness(ctx context.Context, t *testing.T, client Manager, dir string) scripting.Harness {
	opts := testutil.ValidGolangScriptingHarnessOptions(dir)
	sh, err := client.CreateScripting(ctx, opts)
	require.NoError(t, err)
	return sh
}
