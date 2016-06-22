package operations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mongodb/curator/repobuilder"
	"github.com/tychoish/grip"
	"github.com/urfave/cli"
)

func Repo() cli.Command {
	return cli.Command{
		Name:  "repo",
		Usage: "build repository",
		Flags: repoFlags(),
		Action: func(c *cli.Context) error {
			return buildRepo(
				c.String("packages"),
				c.String("config"),
				c.String("distro"),
				c.String("edition"),
				c.String("version"),
				c.String("arch"),
				c.String("profile"),
				c.Bool("dry-run"))
		},
	}
}

func repoFlags() []cli.Flag {
	confPath, err := filepath.Abs("repo_config.yaml")
	grip.CatchErrorFatal(err)

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	return []cli.Flag{
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
			Name:  "packages",
			Usage: "path to packages",
		},
		cli.StringFlag{
			Name:  "profile",
			Usage: "aws profile",
			Value: profile,
		},
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "make task operate in a dry-run mode",
		},
	}
}

func buildRepo(packages, configPath, distro, edition, version, arch, profile string, dryRun bool) error {
	pkgs, err := filepath.Glob(packages)
	if err != nil {
		grip.CatchError(err)
		return err
	}

	if len(pkgs) == 0 {
		e := fmt.Sprintf("there are no packages in '%s'", packages)
		grip.Error(e)
		return errors.New(e)
	}

	conf, err := repobuilder.GetConfig(configPath)
	if err != nil {
		grip.CatchError(err)
		return err
	}

	repo, ok := conf.GetRepositoryDefinition(distro, edition)
	if !ok {
		e := fmt.Sprintf("repo not defined for distro=%s, edition=%s ", distro, edition)
		grip.Error(e)
		return errors.New(e)
	}

	if repo.Type == repobuilder.RPM {
		job, err := repobuilder.NewBuildRPMRepo(repo, version, arch, profile, pkgs...)
		job.DryRun = dryRun
		if err != nil {
			return err
		}
		return job.Run()
	} else if repo.Type == repobuilder.DEB {
		// job, err := repobuilder.NewBuildDEBRepo(repo, version, arch, pkgs...)
		return errors.New("deb repositories are not yet supported")
	} else {
		return fmt.Errorf("%s repositories are not supported", repo.Type)
	}
}
