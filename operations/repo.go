package operations

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/mongodb/curator"
	"github.com/mongodb/curator/barquesubmit"
	"github.com/mongodb/curator/repobuilder"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func repoSubmit() cli.Command {
	return cli.Command{
		Name:  "submit",
		Usage: "submit a repobuilder job to a remote service",
		Flags: repoFlags(
			cli.StringSliceFlag{
				Name:  "packages",
				Usage: "package filepaths",
			},
			cli.StringFlag{
				Name:  "service",
				Usage: "specify the path to a repobuilder service",
				Value: "https://barque.mongodb.com/",
			},
			cli.StringFlag{
				Name:   "username",
				Usage:  "specify the username for a user to authenticate to the repobuilding service",
				EnvVar: "BARQUE_USERNAME",
			},
			cli.StringFlag{
				Name:   "password",
				Usage:  "specify the password to authenticate to the repobuilding service",
				EnvVar: "BARQUE_PASSWORD",
			},
			cli.StringFlag{
				Name:   "api_key",
				Usage:  "specify the API key to authenticate to the repobuilding service",
				EnvVar: "BARQUE_API_KEY",
			},
			cli.StringFlag{
				Name:  "notary_key_name_env",
				Usage: "notary key name environment variable name",
				Value: "NOTARY_KEY_NAME",
			},
			cli.StringFlag{
				Name:  "notary_token_env",
				Usage: "notary token environment variable name",
				Value: "NOTARY_TOKEN",
			},
		),
		Action: func(c *cli.Context) error {
			ctx, cancel := ctxWithTimeout(c.Duration("timeout"))
			defer cancel()

			grip.Infof("curator version: %s", curator.BuildRevision)

			return submitRepo(
				ctx,
				submitRepoOptions{
					url:              c.String("service"),
					username:         c.String("username"),
					password:         c.String("password"),
					apiKey:           c.String("api_key"),
					configPath:       c.String("config"),
					distro:           c.String("distro"),
					edition:          c.String("edition"),
					version:          c.String("version"),
					arch:             c.String("arch"),
					packages:         c.StringSlice("packages"),
					notaryKeyNameEnv: c.String("notary_key_name_env"),
					notaryTokenEnv:   c.String("notary_token_env"),
				},
			)
		},
	}

}

func repoFlags(flags ...cli.Flag) []cli.Flag {
	confPath, err := filepath.Abs("repo_config.yaml")
	grip.EmergencyFatal(err)

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	return append([]cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: confPath,
			Usage: "path of a curator repository configuration file",
		},
		cli.StringFlag{
			Name:  "distro",
			Usage: "short name of a distro",
		},
		cli.StringFlag{
			Name:  "edition",
			Usage: "build edition",
		},
		cli.StringFlag{
			Name:  "version",
			Usage: "a mongodb version",
		},
		cli.StringFlag{
			Name:  "arch",
			Usage: "target architecture of package",
		},
		cli.StringFlag{
			Name:  "profile",
			Usage: "aws profile",
			Value: profile,
		},
		cli.DurationFlag{
			Name:  "timeout",
			Usage: "specify a timeout for operations. Defaults to unlimited timeout if not specified",
		},
	}, flags...)
}

type submitRepoOptions struct {
	url              string
	username         string
	password         string
	apiKey           string
	configPath       string
	distro           string
	edition          string
	version          string
	arch             string
	packages         []string
	notaryKeyNameEnv string
	notaryTokenEnv   string
}

func submitRepo(ctx context.Context, opts submitRepoOptions) error {
	// validate inputs
	if opts.edition == "community" {
		opts.edition = "org"
	}

	// get configuration objects.
	conf, err := repobuilder.GetConfig(opts.configPath)
	if err != nil {
		grip.Error(err)
		return errors.Wrap(err, "problem getting repo config")
	}

	repo, ok := conf.GetRepositoryDefinition(opts.distro, opts.edition)
	if !ok {
		e := fmt.Sprintf("repo not defined for distro=%s, edition=%s ", opts.distro, opts.edition)
		grip.Error(e)
		return errors.New(e)
	}

	jobOpts := repobuilder.JobOptions{
		Configuration: conf,
		Distro:        repo,
		Version:       opts.version,
		Arch:          opts.arch,
		Packages:      opts.packages,
		NotaryKey:     os.Getenv(opts.notaryKeyNameEnv),
		NotaryToken:   os.Getenv(opts.notaryTokenEnv),
		JobID:         uuid.New().String(),
	}

	client, err := barquesubmit.New(opts.url)
	if err != nil {
		return errors.Wrap(err, "problem constructing barque client")
	}

	if opts.username != "" && opts.apiKey != "" {
		client.SetCredentials(opts.username, opts.apiKey)
	} else if err = client.Login(ctx, opts.username, opts.password); err != nil {
		return errors.Wrap(err, "problem authenticating to barque")
	}

	id, err := client.SubmitJob(ctx, jobOpts)
	if err != nil {
		return errors.Wrap(err, "problem submitting repobuilder job")
	}

	startAt := time.Now()
	checks := 0
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
RETRY:
	for {
		checks++
		select {
		case <-ctx.Done():
			return errors.New("operation timed out")
		case <-timer.C:
			stat, err := client.CheckJobStatus(ctx, id)
			if err != nil {
				grip.Error(err)
				return errors.Wrap(err, "problem checking job status")
			}

			if !stat.Status.Completed {
				grip.Info(message.Fields{
					"job":               stat.ID,
					"wallclock_seconds": time.Since(startAt).Seconds(),
					"duration_seconds":  time.Since(stat.Timing.Start).Seconds(),
					"in_progress":       stat.Status.InProgress,
					"complete":          stat.Status.Completed,
					"checks":            checks,
				})
				timer.Reset(30*time.Second + time.Duration(rand.Int63n(int64(time.Minute))))
				continue RETRY
			}

			if stat.HasErrors {
				grip.Error(message.Fields{
					"job":               stat.ID,
					"wallclock_seconds": time.Since(startAt).Seconds(),
					"duration_seconds":  stat.Timing.Duration().Seconds(),
					"errors":            stat.Status.Errors,
					"checks":            checks,
				})

				return errors.Errorf("job '%s' completed with error [%s]", id, stat.Error)
			}

			grip.Info(message.Fields{
				"job":               stat.ID,
				"duration_seconds":  stat.Timing.Duration().Seconds(),
				"wallclock_seconds": time.Since(startAt).Seconds(),
				"complete":          stat.Status.Completed,
				"in_progress":       stat.Status.InProgress,
				"checks":            checks,
			})
			return nil
		}
	}
}
