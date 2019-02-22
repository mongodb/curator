package operations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mongodb/curator/repobuilder"
	"github.com/mongodb/grip"
	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
)

// Index returns the index page rebuilder command line interface.
func Index() cli.Command {
	confPath, err := filepath.Abs("repo_config.yaml")
	grip.CatchEmergencyFatal(err)

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	pwd, err := os.Getwd()
	grip.CatchEmergencyFatal(err)
	workingDir := filepath.Join(pwd, uuid.Must(uuid.NewV4()).String())

	return cli.Command{
		Name:  "rebuild-index-pages",
		Usage: "rebuild index.html pages for a bucket.",
		Flags: []cli.Flag{
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
				Name:  "profile",
				Usage: "aws profile",
				Value: profile,
			},
			cli.BoolFlag{
				Name: "dry-run",
				Usage: fmt.Sprintln("task runs in a dry-run mode.",
					"files are downloaded but no files are uploaded."),
			},
			cli.StringFlag{
				Name: "name",
				Usage: fmt.Sprintln("the public name of the repo for use in",
					"footer. Defaults to the name of the s3 bucket."),
			},
			cli.StringFlag{
				Name: "distro",
				Usage: fmt.Sprintln("short name of a distro from the config.",
					"only used to get a bucket name: all index pages",
					"for the bucket are rebuilt"),
			},
			cli.StringFlag{
				Name:  "edition",
				Usage: "build edition",
			},
		},
		Action: func(c *cli.Context) error {
			return rebuildIndexPages(
				c.String("config"),
				c.String("dir"),
				c.String("name"),
				c.String("distro"),
				c.String("edition"),
				c.Bool("dry-run"))
		},
	}
}

func rebuildIndexPages(configPath, dir, name, distro, edition string, dryRun bool) error {
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

	j := repobuilder.NewIndexBuildJob(conf, dir, name, repo.Bucket, dryRun)

	j.Run(context.TODO())

	return j.Error()
}
