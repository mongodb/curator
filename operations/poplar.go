package operations

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/evergreen-ci/aviation"
	"github.com/evergreen-ci/poplar"
	"github.com/evergreen-ci/poplar/rpc"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/recovery"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

const (
	envVarPoplarRecorderRPCPort  = "POPLAR_RPC_PORT"
	envVarPoplarRecorderRPCHost  = "POPLAR_RPC_HOST"
	defaultPoplarRecorderRPCPort = 2288
	defaultLocalHostName         = "localhost"
)

// Poplar command line function.
func Poplar() cli.Command {
	return cli.Command{
		Name:  "poplar",
		Usage: "a performance testing and metrics reporting toolkit",
		Flags: []cli.Flag{},
		Subcommands: []cli.Command{
			poplarReport(),
			poplarGRPC(),
		},
	}
}

func poplarGRPC() cli.Command {
	return cli.Command{
		Name:  "grpc",
		Usage: "run an RPC service for accumulating raw event payloads",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "port",
				EnvVar: envVarPoplarRecorderRPCPort,
				Value:  defaultPoplarRecorderRPCPort,
			},
			cli.StringFlag{
				Name:   "host",
				EnvVar: envVarPoplarRecorderRPCHost,
				Value:  defaultLocalHostName,
			},
		},
		Action: func(c *cli.Context) error {
			port := c.Int("port")
			host := c.String("host")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			addr := fmt.Sprintf("%s:%d", host, port)
			lis, err := net.Listen("tcp", addr)
			if err != nil {
				return errors.WithStack(err)
			}

			grip.Infof("starting poplar gRPC service at '%s'", addr)
			rpcSrv := grpc.NewServer()

			registry := poplar.NewRegistry()
			if err = rpc.AttachService(registry, rpcSrv); err != nil {
				return errors.Wrap(err, "building service")
			}

			go signalListener(ctx, cancel)
			go func() { grip.Warning(rpcSrv.Serve(lis)) }()

			wait := make(chan struct{})

			go func() {
				defer close(wait)
				defer recovery.LogStackTraceAndContinue("waiting for rpc service")
				<-ctx.Done()
				rpcSrv.Stop()
				grip.Info("poplar rpc service terminated")
			}()

			<-wait
			return nil
		},
	}
}

func poplarReport() cli.Command {
	const (
		serviceFlagName           = "service"
		pathFlagName              = "path"
		insecureFlagName          = "insecure"
		caFileFlagName            = "ca"
		certFileFlagName          = "cert"
		keyFileFlagName           = "key"
		apiUsernameFlagName       = "api-username"
		apiKeyFlagName            = "api-key"
		apiUsernameHeaderFlagName = "api-username-header"
		apiKeyHeaderFlagName      = "api-key-header"
		awsAccessKeyName          = "aws-access-keys"
		awsSecretKeyName          = "aws-secret-key"
        awsTokenName              = "aws-token"
		ResultsHandlerHostName    = "results-handler-host"
		dryRunFlagName            = "dry-run"
		dryRunFlagNameShort       = "n"
	)

	return cli.Command{
		Name:  "send",
		Usage: "send a metrics report",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  serviceFlagName,
				Usage: "specify the address of the metrics service",
			},
			cli.BoolFlag{
				Name:  insecureFlagName,
				Usage: "disables certificate validation requirements",
			},
			cli.StringFlag{
				Name:  certFileFlagName,
				Usage: "specify the client certificate to connect over TLS",
			},
			cli.StringFlag{
				Name:  caFileFlagName,
				Usage: "specify the client ca to connect over TLS",
			},
			cli.StringFlag{
				Name:  keyFileFlagName,
				Usage: "specify the client cert key to connect over TLS",
			},
			cli.StringFlag{
				Name:  apiUsernameFlagName,
				Usage: "specify the username for API authentication",
			},
			cli.StringFlag{
				Name:  apiKeyFlagName,
				Usage: "specify the API key for API authentication",
			},
			cli.StringFlag{
				Name:  apiUsernameHeaderFlagName,
				Usage: "specify the username header for API authentication",
			},
			cli.StringFlag{
				Name:  apiKeyHeaderFlagName,
				Usage: "specify the API key header for API authentication",
			},
			cli.StringFlag{
				Name:  awsAccessKeyName,
				Usage: "AWS access key ID to upload results to Data Pipes",
			},
			cli.StringFlag{
				Name:  awsSecretKeyName,
				Usage: "AWS secret key to upload results to Data Pipes",
			},
            cli.StringFlag{
				Name:  awsTokenName,
				Usage: "AWS token to upload results to Data Pipes",
			}
			cli.StringFlag{
				Name:  pathFlagName,
				Usage: "specify the path of the input file, may be the first positional argument",
			},
			cli.StringFlag{
				Name:  ResultsHandlerHostName,
				Usage: "specify the host URL to upload results to PSS",
			},
			cli.BoolFlag{
				Name:  dryRunFlagName + "," + dryRunFlagNameShort,
				Usage: "enables dry run",
			},
		},
		Before: mergeBeforeFuncs(
			requireStringFlag(serviceFlagName),
			requireFileOrPositional(pathFlagName),
		),
		Action: func(c *cli.Context) error {
			addr := c.String(serviceFlagName)
			fileName := c.String(pathFlagName)
			isInsecure := c.Bool(insecureFlagName)
			caFile := c.String(caFileFlagName)
			certFile := c.String(certFileFlagName)
			keyFile := c.String(keyFileFlagName)
			apiUsername := c.String(apiUsernameFlagName)
			apiKey := c.String(apiKeyFlagName)
			apiUsernameHeader := c.String(apiUsernameHeaderFlagName)
			apiKeyHeader := c.String(apiKeyHeaderFlagName)
			awsAccessKey := c.String(awsAccessKeyName)
			awsSecretKey := c.String(awsSecretKeyName)
            awsToken := c.String(awsTokenName)
			resultsHandlerHost := c.String(ResultsHandlerHostName)
			dryRun := c.Bool("dry-run") || c.Bool("n")

			report, err := poplar.LoadReport(fileName)
			if err != nil {
				return errors.WithStack(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var tlsConf *tls.Config
			if !isInsecure {
				if certFile != "" {
					tlsConf, err = aviation.GetClientTLSConfigFromFiles([]string{caFile}, certFile, keyFile)
					if err != nil {
						return errors.WithStack(err)
					}
				} else {
					cp, err := aviation.GetCACertPool()
					if err != nil {
						return errors.WithStack(err)
					}
					tlsConf = &tls.Config{RootCAs: cp}
				}
			}
			conn, err := aviation.Dial(ctx, aviation.DialOptions{
				Address:       addr,
				Retries:       10,
				TLSConf:       tlsConf,
				Username:      apiUsername,
				APIKey:        apiKey,
				APIUserHeader: apiUsernameHeader,
				APIKeyHeader:  apiKeyHeader,
			})
			if err != nil {
				return errors.WithStack(err)
			}

			opts := rpc.UploadReportOptions{
				Report:             report,
				ClientConn:         conn,
				DryRun:             dryRun,
				AWSSecretKey:       awsSecretKey,
				AWSAccessKey:       awsAccessKey,
				AwsToken:           awsToken,
				ResultsHandlerHost: resultsHandlerHost,
				SerializeUpload:    true,
			}
			if err := rpc.UploadReport(ctx, opts); err != nil {
				return errors.WithStack(err)
			}

			return nil
		},
	}
}
