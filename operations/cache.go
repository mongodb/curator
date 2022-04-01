package operations

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/evergreen-ci/lru"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// PruneCache returns a command object for cache pruning helper for
// removing objects from a file system cache based on last
// modification time.
func PruneCache() cli.Command {
	return cli.Command{
		Name:  "prune",
		Usage: "prunes contents of a filesystem based on modification time. Follows symlinks.",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "max-size",
				Usage: "specify the max size of the cache to prune to in megabytes",
			},
			cli.StringFlag{
				Name:   "path",
				EnvVar: "CURATOR_ARTIFACTS_DIRECTORY",
				Value:  filepath.Join(os.TempDir(), "curator-artifact-cache"),
				Usage:  "path to top level of cache directory",
			},
			cli.BoolFlag{
				Name: "recursive",
				Usage: fmt.Sprintln("when specified, skips directories (and cleans them up later)",
					"and examines each file object independently throughout the tree."),
			},
			cli.BoolFlag{
				Name:  "dry-run",
				Usage: "dry run mode does not remove files from the file system.",
			},
		},
		Action: func(c *cli.Context) error {
			maxSize := c.Int("max-size") * 1024 * 1024
			return pruneCache(c.String("path"), maxSize, c.Bool("recursive"), c.Bool("dry-run"))
		},
	}
}

func pruneCache(path string, maxSize int, recursive, dryRun bool) error {
	var cache *lru.Cache
	var err error

	if recursive {
		cache, err = lru.TreeContents(path)
	} else {
		cache, err = lru.DirectoryContents(path, false)
	}

	if err != nil {
		return errors.Wrapf(err, "building cache for '%s'", path)
	}

	return errors.Wrap(cache.Prune(maxSize, []string{"full.json"}, dryRun), "pruning cache")
}
