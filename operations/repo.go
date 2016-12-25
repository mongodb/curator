package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mongodb/curator/repobuilder"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/tychoish/grip"
	"github.com/urfave/cli"
)

// Repo returns a cli.Command object for the repo building and
// rebuilding operation.
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
				c.Bool("dry-run"),
				c.Bool("rebuild"))
		},
	}
}

func repoFlags() []cli.Flag {
	confPath, err := filepath.Abs("repo_config.yaml")
	grip.CatchEmergencyFatal(err)

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	pwd, err := os.Getwd()
	grip.CatchEmergencyFatal(err)
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
		cli.BoolFlag{
			Name:  "rebuild",
			Usage: "rebuild a repository without adding any new packages",
		},
	}
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

func buildRepo(packages, configPath, workingDir, distro, edition, version, arch, profile string, dryRun, rebuild bool) error {
	// validate inputs
	if edition == "community" {
		edition = "org"
	}

	// get configuration objects.
	conf, err := repobuilder.GetConfig(configPath)
	if err != nil {
		grip.CatchError(err)
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

	job, err := repobuilder.NewBuildRepoJob(conf, repo, version, arch, profile, pkgs...)
	if err != nil {
		return errors.Wrap(err, "problem constructing task for building repository")
	}
	job.WorkSpace = workingDir
	job.DryRun = dryRun

	job.Run()
	err = job.Error()
	if err != nil {
		return errors.Wrap(err, "encountered error rebuilding repository")
	}

	return nil
}
