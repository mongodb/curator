package operations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mongodb/curator/repobuilder"
	"github.com/satori/go.uuid"
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
				c.String("dir"),
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

	pwd, err := os.Getwd()
	grip.CatchErrorFatal(err)
	workingDir := filepath.Join(pwd, uuid.NewV4().String())

	return []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: confPath,
			Usage: "path of a curator repository configuration file",
		},
		cli.StringFlag{
			Name:  "dir",
			Value: workingDir,
			Usage: "path to a workspace for curator to do its work",
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
			Usage: "path to packages, searches for valid packages recursively",
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

func getPackages(rootPath, suffix string) ([]string, error) {
	var output []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
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
		return []string{}, fmt.Errorf("no packages found in path '%s'", rootPath)
	}

	return output, err
}

func buildRepo(packages, configPath, workingDir, distro, edition, version, arch, profile string, dryRun bool) error {
	// validate inputs
	if edition == "community" {
		edition = "org"
	}

	// get configuration objects.
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

	// build the packages:
	if repo.Type == repobuilder.RPM {
		pkgs, err := getPackages(packages, ".rpm")
		if err != nil {
			grip.CatchError(err)
			return err
		}

		job, err := repobuilder.NewBuildRPMRepo(conf, repo, version, arch, profile, pkgs...)
		if err != nil {
			return err
		}
		job.WorkSpace = workingDir
		job.DryRun = dryRun
		return job.Run()
	} else if repo.Type == repobuilder.DEB {
		pkgs, err := getPackages(packages, ".deb")
		if err != nil {
			grip.CatchError(err)
			return err
		}

		job, err := repobuilder.NewBuildDEBRepo(conf, repo, version, arch, profile, pkgs...)
		if err != nil {
			return err
		}
		job.WorkSpace = workingDir
		job.DryRun = dryRun
		return job.Run()
	} else {
		return fmt.Errorf("%s repositories are not supported", repo.Type)
	}
}
