package operations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/evergreen-ci/bond"
	"github.com/evergreen-ci/bond/recall"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Artifacts returns a command object for the "archives" sub command
// which contains functions to download MongoDB archives.
func Artifacts() cli.Command {
	return cli.Command{
		Name:    "artifacts",
		Aliases: []string{"archives", "builds"},
		Usage:   "download ",
		Subcommands: []cli.Command{
			{
				Name:    "download",
				Usage:   "downloads builds of MongoDB",
				Aliases: []string{"dl", "get"},
				Flags: buildInfoFlags(baseDlFlags(true,
					cli.StringFlag{
						Name:  "timeout",
						Value: "no-timeout",
						Usage: "maximum duration for operation, defaults to no time out",
					})...),
				Action: func(c *cli.Context) error {
					var cancel context.CancelFunc
					ctx := context.Background()

					timeout := c.String("timeout")
					if timeout != "no-timeout" {
						ttl, err := time.ParseDuration(timeout)
						if err != nil {
							return errors.Wrapf(err, "%s is not a valid timeout", timeout)
						}
						ctx, cancel = context.WithTimeout(ctx, ttl)
						defer cancel()
					} else {
						ctx, cancel = context.WithCancel(ctx)
						defer cancel()
					}

					target := c.String("target")
					if strings.Contains(target, "auto") {
						target = bond.GetTargetDistro()
					}

					opts := bond.BuildOptions{
						Target:  target,
						Arch:    bond.MongoDBArch(c.String("arch")),
						Edition: bond.MongoDBEdition(c.String("edition")),
						Debug:   c.Bool("debug"),
					}

					err := recall.FetchReleases(ctx, c.StringSlice("version"), c.String("path"), opts)
					if err != nil {
						return errors.Wrap(err, "problem fetching releases")
					}

					return nil
				},
			},
			{
				Name:  "list-variants",
				Usage: "find all targets, editions and architectures for a version",
				Flags: baseDlFlags(false),
				Action: func(c *cli.Context) error {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					version, err := getVersionForListing(ctx, c.String("version"), c.String("path"))
					if err != nil {
						return errors.Wrap(err, "problem fetching version")
					}

					fmt.Println(version.GetBuildTypes())
					return nil
				},
			},
			{
				Name:  "list-map",
				Usage: "find targets/edition/architecture mappings for a version",
				Flags: baseDlFlags(false),
				Action: func(c *cli.Context) error {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					version, err := getVersionForListing(ctx, c.String("version"), c.String("path"))
					if err != nil {
						return errors.Wrap(err, "problem fetching version")
					}

					fmt.Println(version)
					return nil
				},
			},
			{
				Name:  "list-all",
				Usage: "prints a listing of the current contents of the version cache",
				Flags: baseDlFlags(false),
				Action: func(c *cli.Context) error {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					catalog, err := bond.NewCatalog(ctx, c.String("path"))
					if err != nil {
						return errors.Wrap(err, "problem building catalog")
					}
					fmt.Println(catalog)
					return nil
				},
			},
			{
				Name:  "get-path",
				Usage: "get path to a build",
				Flags: buildInfoFlags(baseDlFlags(false)...),
				Action: func(c *cli.Context) error {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					catalog, err := bond.NewCatalog(ctx, c.String("path"))
					if err != nil {
						return errors.Wrap(err, "problem building catalog")
					}

					path, err := catalog.Get(c.String("version"), c.String("edition"),
						c.String("target"), c.String("arch"), c.Bool("debug"))

					if err != nil {
						return errors.Wrap(err, "problem finding build")
					}

					fmt.Println(path)

					return nil
				},
			},
		},
	}
}

func buildInfoFlags(flags ...cli.Flag) []cli.Flag {
	var target string
	var arch string

	if runtime.GOOS == "darwin" {
		target = "osx"
	} else {
		target = runtime.GOOS
	}

	if runtime.GOARCH == "amd64" {
		arch = "x86_64"
	} else if runtime.GOARCH == "386" {
		arch = "i686"
	} else if runtime.GOARCH == "arm" {
		arch = "arm64"
	} else {
		arch = runtime.GOARCH
	}

	return append(flags,
		cli.StringFlag{
			Name:  "target",
			Value: target,
			Usage: "name of target platform or operating system",
		},
		cli.StringFlag{
			Name:  "arch",
			Value: arch,
			Usage: "name of target architecture",
		},
		cli.StringFlag{
			Name:  "edition",
			Value: "base",
			Usage: "name of build edition",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "specify to download debug symbols",
		})
}

func baseDlFlags(versionSlice bool, flags ...cli.Flag) []cli.Flag {
	if versionSlice {
		flags = append(flags,
			cli.StringSliceFlag{
				Name:  "version",
				Usage: "specify a version (may specify multiple times)",
			})
	} else {
		flags = append(flags,
			cli.StringFlag{
				Name:  "version",
				Usage: "specify a version",
			})
	}

	return append(flags,
		cli.StringFlag{
			Name:   "path",
			EnvVar: "CURATOR_ARTIFACTS_DIRECTORY",
			Value:  filepath.Join(os.TempDir(), "curator-artifact-cache"),
			Usage:  "path to top level of cache directory",
		})
}

func getVersionForListing(ctx context.Context, release, path string) (*bond.ArtifactVersion, error) {
	feed, err := bond.GetArtifactsFeed(ctx, path)
	if err != nil {
		return nil, errors.Wrap(err, "problem fetching artifacts feed")
	}

	version, ok := feed.GetVersion(release)
	if !ok {
		return nil, errors.Errorf("no version for %s", release)
	}

	return version, nil
}
