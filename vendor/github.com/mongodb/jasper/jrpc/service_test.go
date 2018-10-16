package jrpc

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mongodb/jasper"
	"github.com/mongodb/jasper/jrpc/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestJRPCService(t *testing.T) {
	for managerName, makeManager := range map[string]func() jasper.Manager{
		"Blocking":    jasper.NewLocalManagerBlockingProcesses,
		"Nonblocking": jasper.NewLocalManager,
	} {
		t.Run(managerName, func(t *testing.T) {
			for testName, testCase := range map[string]func(context.Context, *testing.T, jasper.CreateOptions, internal.JasperProcessManagerClient, string, string){
				"CreateWithLogFile": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					file, err := ioutil.TempFile(buildDir, "out.txt")
					require.NoError(t, err)
					defer os.Remove(file.Name())

					logger := jasper.Logger{
						Type: jasper.LogFile,
						Options: jasper.LogOptions{
							FileName: file.Name(),
							Format:   jasper.LogFormatPlain,
						},
					}
					opts.Output.Loggers = []jasper.Logger{logger}

					procInfo, err := client.Create(ctx, internal.ConvertCreateOptions(&opts))
					require.NoError(t, err)
					require.NotNil(t, procInfo)

					outcome, err := client.Wait(ctx, &internal.JasperProcessID{Value: procInfo.Id})
					require.NoError(t, err)
					require.True(t, outcome.Success)

					info, err := os.Stat(file.Name())
					require.NoError(t, err)
					assert.NotZero(t, info.Size())

					fileContents, err := ioutil.ReadFile(file.Name())
					require.NoError(t, err)
					assert.Contains(t, string(fileContents), output)
				},
				"DownloadFileCreatesResource": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					file, err := ioutil.TempFile(buildDir, "out.txt")
					require.NoError(t, err)
					defer os.Remove(file.Name())

					info := jasper.DownloadInfo{
						URL:  "http://example.com",
						Path: file.Name(),
					}
					outcome, err := client.DownloadFile(ctx, internal.ConvertDownloadInfo(info))
					require.NoError(t, err)
					assert.True(t, outcome.Success)

					fileInfo, err := os.Stat(file.Name())
					require.NoError(t, err)
					assert.NotZero(t, fileInfo.Size())
				},
				"DownloadFileFailsForInvalidArchiveFormat": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					fileName := filepath.Join(buildDir, "out.txt")

					info := jasper.DownloadInfo{
						URL:  "https://example.com",
						Path: fileName,
						ArchiveOpts: jasper.ArchiveOptions{
							ShouldExtract: true,
							Format:        jasper.ArchiveFormat("foo"),
						},
					}
					_, err := client.DownloadFile(ctx, internal.ConvertDownloadInfo(info))
					assert.Error(t, err)
				},
				"DownloadFileFailsForInvalidURL": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					fileName := filepath.Join(buildDir, "out.txt")

					info := jasper.DownloadInfo{
						URL:  "://example.com",
						Path: fileName,
					}
					_, err := client.DownloadFile(ctx, internal.ConvertDownloadInfo(info))
					assert.Error(t, err)
				},
				"DownloadFileFailsForNonexistentURL": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					fileName := filepath.Join(buildDir, "out.txt")

					info := jasper.DownloadInfo{
						URL:  "http://example.com/foo",
						Path: fileName,
					}
					_, err := client.DownloadFile(ctx, internal.ConvertDownloadInfo(info))
					assert.Error(t, err)
				},
				"DownloadFileAsyncPassesWithValidInfo": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					file, err := ioutil.TempFile(buildDir, "out.txt")
					require.NoError(t, err)
					defer os.Remove(file.Name())

					info := jasper.DownloadInfo{
						URL:  "http://example.com",
						Path: file.Name(),
					}
					outcome, err := client.DownloadFileAsync(ctx, internal.ConvertDownloadInfo(info))
					require.NoError(t, err)
					assert.True(t, outcome.Success)

				waitAsyncDownload:
					for {
						select {
						case <-ctx.Done():
							assert.Fail(t, "asynchronous download did not complete before context deadline exceeded")
						default:
							fileInfo, err := os.Stat(file.Name())
							require.NoError(t, err)
							if fileInfo.Size() != 0 {
								break waitAsyncDownload
							}
						}
					}
				},
				"DownloadFileAsyncFailsForInvalidArchiveFormat": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					fileName := filepath.Join(buildDir, "out.txt")

					info := jasper.DownloadInfo{
						URL:  "https://example.com",
						Path: fileName,
						ArchiveOpts: jasper.ArchiveOptions{
							ShouldExtract: true,
							Format:        jasper.ArchiveFormat("foo"),
						},
					}
					_, err := client.DownloadFileAsync(ctx, internal.ConvertDownloadInfo(info))
					assert.Error(t, err)
				},
				"DownloadFileAsyncFailsForInvalidURL": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					fileName := filepath.Join(buildDir, "out.txt")

					info := jasper.DownloadInfo{
						URL:  "://example.com",
						Path: fileName,
					}
					_, err := client.DownloadFileAsync(ctx, internal.ConvertDownloadInfo(info))
					assert.Error(t, err)
				},
				"GetBuildloggerURLsFailsWithNonexistentProcess": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					urls, err := client.GetBuildloggerURLs(ctx, &internal.JasperProcessID{Value: "foo"})
					assert.Error(t, err)
					assert.Nil(t, urls)
				},
				"GetBuildloggerURLsFailsWithoutBuildlogger": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string, buildDir string) {
					opts.Output.Loggers = []jasper.Logger{jasper.Logger{Type: jasper.LogDefault, Options: jasper.LogOptions{Format: jasper.LogFormatPlain}}}
					info, err := client.Create(ctx, internal.ConvertCreateOptions(&opts))
					assert.NoError(t, err)

					urls, err := client.GetBuildloggerURLs(ctx, &internal.JasperProcessID{Value: info.Id})
					assert.Error(t, err)
					assert.Nil(t, urls)
				},
				//"": func(ctx context.Context, t *testing.T, opts jasper.CreateOptions, client internal.JasperProcessManagerClient, output string) {},
			} {
				t.Run(testName, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), taskTimeout)
					defer cancel()
					output := "foobar"
					opts := jasper.CreateOptions{Args: []string{"echo", output}}

					manager := makeManager()
					addr, err := startJRPC(ctx, manager)
					require.NoError(t, err)

					conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
					require.NoError(t, err)
					client := internal.NewJasperProcessManagerClient(conn)

					go func() {
						<-ctx.Done()
						conn.Close()
					}()

					cwd, err := os.Getwd()
					require.NoError(t, err)
					buildDir := filepath.Join(filepath.Dir(cwd), "build")
					absBuildDir, err := filepath.Abs(buildDir)
					require.NoError(t, err)

					testCase(ctx, t, opts, client, output, absBuildDir)
				})
			}
		})
	}
}
