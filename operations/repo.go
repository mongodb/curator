package operations

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
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

// Repo returns a cli.Command object for the repo building and
// rebuilding operation.
func Repo() cli.Command {
	pwd, err := os.Getwd()
	grip.EmergencyFatal(err)
	workingDir := filepath.Join(pwd, uuid.New().String())

	return cli.Command{
		Name:  "repo",
		Usage: "build repository",
		Flags: repoFlags(
			cli.StringFlag{
				Name:  "packages",
				Usage: "path to packages, searches for valid packages recursively",
			},
			cli.StringFlag{
				Name:  "dir",
				Value: workingDir,
				Usage: "path to a workspace for curator to do its work",
			},
			cli.BoolFlag{
				Name:  "dry-run",
				Usage: "make task operate in a dry-run mode",
			},
			cli.BoolFlag{
				Name:  "verbose",
				Usage: "run task in verbose (debug) mode",
			},
			cli.BoolFlag{
				Name:  "rebuild",
				Usage: "rebuild a repository without adding any new packages",
			},
			cli.IntFlag{
				Name:  "retries",
				Usage: "number of times to retry in the case of failures",
				Value: 1,
			},
		),
		Subcommands: []cli.Command{
			repoSubmit(),
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := ctxWithTimeout(c.Duration("timeout"))
			defer cancel()

			grip.Infof("curator version: %s", curator.BuildRevision)

			return buildRepo(ctx,
				c.String("packages"),
				c.String("config"),
				c.String("dir"),
				c.String("distro"),
				c.String("edition"),
				c.String("version"),
				c.String("arch"),
				c.String("prqofile"),
				c.Bool("dry-run"),
				c.Bool("verbose"),
				c.Bool("rebuild"),
				c.Int("retries"),
			)
		},
	}
}

func repoSubmit() cli.Command {
	return cli.Command{
		Name:  "submit",
		Usage: "submit a repobuilder job to a remote service",
		Flags: repoFlags(
			cli.StringSliceFlag{
				Name:  "packages",
				Usage: "path to packages, searches for valid packages recursively",
			},

			cli.StringFlag{
				Name:  "service",
				Value: "https://barque.mongodb.comm",
			},
			cli.StringFlag{
				Name:   "username",
				EnvVar: "BARQUE_USERNAME",
			},
			cli.StringFlag{
				Name:   "password",
				EnvVar: "BARQUE_PASSWORD",
			},
		),
		Subcommands: []cli.Command{},
		Action: func(c *cli.Context) error {
			ctx, cancel := ctxWithTimeout(c.Duration("timeout"))
			defer cancel()

			grip.Infof("curator version: %s", curator.BuildRevision)

			return submitRepo(ctx,
				barqueServiceInfo{
					url:      c.String("service"),
					username: c.String("username"),
					password: c.String("password"),
				},
				c.String("config"),
				c.String("distro"),
				c.String("edition"),
				c.String("version"),
				c.String("arch"),
				c.StringSlice("packages"))
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

func getPackages(rootPath, suffix string) ([]string, error) {
	var output []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(info.Name(), suffix) {
			output = append(output, path)
		}

		return nil
	})

	if err != nil {
		return []string{}, err
	}

	if len(output) == 0 {
		return []string{}, fmt.Errorf("no '%s' packages found in path '%s'", suffix, rootPath)
	}

	return output, err
}

func buildRepo(ctx context.Context, packages, configPath, workingDir, distro, edition, version, arch, profile string, dryRun, verbose, rebuild bool, retries int) error {
	// validate inputs
	if edition == "community" {
		edition = "org"
	}

	// get configuration objects.
	conf, err := repobuilder.GetConfig(configPath)
	if err != nil {
		grip.Error(err)
		return errors.Wrap(err, "problem getting repo config")
	}

	repo, ok := conf.GetRepositoryDefinition(distro, edition)
	if !ok {
		e := fmt.Sprintf("repo not defined for distro=%s, edition=%s ", distro, edition)
		grip.Error(e)
		return errors.New(e)
	}

	var pkgs []string

	if !rebuild {
		if repo.Type == repobuilder.RPM {
			pkgs, err = getPackages(packages, ".rpm")
		} else if repo.Type == repobuilder.DEB {
			pkgs, err = getPackages(packages, ".deb")
		}

		if err != nil {
			return errors.Wrap(err, "problem finding packages")
		}
	}

	conf.DryRun = dryRun
	conf.Verbose = verbose
	conf.WorkSpace = workingDir

	job, err := repobuilder.NewBuildRepoJob(conf, repo, version, arch, profile, pkgs...)
	if err != nil {
		return errors.Wrap(err, "problem constructing task for building repository")
	}

	if retries < 1 {
		retries = 1
	}

	catcher := grip.NewCatcher()
	for i := 0; i < retries; i++ {
		job.Run(ctx)
		err = job.Error()
		if err == nil {
			break
		}
		catcher.Add(err)
	}

	return errors.Wrapf(catcher.Resolve(), "encountered problem rebuilding repository after %d retries", retries)
}

type barqueServiceInfo struct {
	url      string
	username string
	password string
}

func submitRepo(ctx context.Context, info barqueServiceInfo, configPath, distro, edition, version, arch string, packages []string) error {
	// validate inputs
	if edition == "community" {
		edition = "org"
	}

	// get configuration objects.
	conf, err := repobuilder.GetConfig(configPath)
	if err != nil {
		grip.Error(err)
		return errors.Wrap(err, "problem getting repo config")
	}

	repo, ok := conf.GetRepositoryDefinition(distro, edition)
	if !ok {
		e := fmt.Sprintf("repo not defined for distro=%s, edition=%s ", distro, edition)
		grip.Error(e)
		return errors.New(e)
	}

	opts := repobuilder.JobOptions{
		Configuration: conf,
		Distro:        repo,
		Version:       version,
		Arch:          arch,
		Packages:      packages,
		JobID:         "tk",
	}

	client, err := barquesubmit.New(info.url)
	if err != nil {
		return errors.Wrap(err, "problem constructing barque client")
	}

	if err = client.Login(ctx, info.username, info.password); err != nil {
		return errors.Wrap(err, "problem authenticating to barque")
	}

	id, err := client.SubmitJob(ctx, opts)
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
				"checks":            checks,
			})
			return nil
		}
	}
}
