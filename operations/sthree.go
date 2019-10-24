/*
S3 File Management

Curator contains simple operations for managing files in S3
Buckets. Supported operations include sync operations, as well as
simple puts, gets, and deletes. The operations are available within the s3
sub-command, for example:

   curator s3 sync-to --jobs <int> --bucket <bucket> --local <path> --prefix <remote>
   curator s3 sync-from --jobs <int> --bucket <bucket> --local <path> --prefix <remote>
   curator s3 delete --bucket <bucket> --name <remote> <, --name <remote>...>
   curator s3 delete-prefix --bucket <bucket> --prefix <remote>
   curator s3 put --bucket <bucket> --file <local> --name <remote>
   curator s3 get --bucket <bucket> --file <local> --name <remote>

For sync commands, the "prefix" argument allows
you to sync only a portion of the bucket (e.g. all items with
key-names that start with that prefix.) For the "push" operation, the
curator prepends prefix (e.g. "folder" or leading ortion of the key
name) to the local file name within the bucket. The prefix need not
end with a "/", though the prefix and filename will be combined with a
"/" character.

Sync operations first compare file names and then compare MD5
checksums, and upload only differing content. Unlike rsync, file sizes
and timestamps are *not* considered. Also there is no "delete" or
"mirror" operation.

Put and get operations perform simple copy operations. You can specify
long path names, with prefix/directories in the remote name.

By default curator attempts to read AWS credentials from the
"AWS_ACCESS_KEY" and "AWS_SECRET_KEY" environment variables (if set),
or the standard "$HOME/.aws/credentials" file or a file specified in
the "AWS_CREDNETIAL_FILE" environment variable. By default curator
reads the "default" profile from the credentials file, but you can
specify a different profile using the "AWS_PROFILE" environment
variable.
*/

package operations

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/evergreen-ci/pail"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	defaultMaxRetries = 20
)

// S3 returns a cli.Command object for the S3 command group which has a
// group of sub-commands.
func S3() cli.Command {
	return cli.Command{
		Name:    "s3",
		Aliases: []string{"sthree"},
		Usage:   "a collection of s3 operations",
		Subcommands: []cli.Command{
			s3PutCmd(),
			s3GetCmd(),
			s3DeleteCmd(),
			s3DeletePrefixCmd(),
			s3DeleteMatchingCmd(),
			s3SyncToCmd(),
			s3SyncFromCmd(),
		},
	}

}

/////////////////////////////////////
//
// Specific S3 Operation Sub-Commands
//
/////////////////////////////////////

func s3PutCmd() cli.Command {
	return cli.Command{
		Name:  "put",
		Usage: "upload a local file object into s3",
		Flags: baseS3Flags(s3opFlags(s3putFlags()...)...),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			opts := pail.S3Options{
				SharedCredentialsProfile: c.String("profile"),
				Region:                   c.String("region"),
				Name:                     c.String("bucket"),
				Permissions:              pail.S3Permissions(c.String("permissions")),
				ContentType:              c.String("type"),
				DryRun:                   c.Bool("dry-run"),
				MaxRetries:               defaultMaxRetries,
			}
			bucket, err := pail.NewS3Bucket(opts)
			if err != nil {
				return errors.Wrap(err, "problem getting new bucket")
			}

			return errors.Wrapf(
				bucket.Upload(ctx, c.String("name"), c.String("file")),
				"problem putting %s in s3",
				c.String("file"),
			)
		},
	}
}

func s3GetCmd() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "download a local file object from s3",
		Flags: baseS3Flags(s3opFlags()...),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			opts := pail.S3Options{
				SharedCredentialsProfile: c.String("profile"),
				Region:                   c.String("region"),
				Name:                     c.String("bucket"),
				DryRun:                   c.Bool("dry-run"),
				MaxRetries:               defaultMaxRetries,
			}
			bucket, err := pail.NewS3Bucket(opts)
			if err != nil {
				return errors.Wrap(err, "problem getting new bucket")
			}

			return errors.Wrapf(
				bucket.Download(ctx, c.String("name"), c.String("file")),
				"problem getting %s from s3",
				c.String("name"),
			)
		},
	}
}

func s3DeleteCmd() cli.Command {
	return cli.Command{
		Name:    "delete",
		Aliases: []string{"del", "rm"},
		Flags: baseS3Flags(
			cli.StringFlag{
				Name:  "name",
				Usage: "the remote s3 resource name, may include the prefix",
			}),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			opts := pail.S3Options{
				SharedCredentialsProfile: c.String("profile"),
				Region:                   c.String("region"),
				Name:                     c.String("bucket"),
				DryRun:                   c.Bool("dry-run"),
				MaxRetries:               defaultMaxRetries,
			}
			bucket, err := pail.NewS3Bucket(opts)
			if err != nil {
				return errors.Wrap(err, "problem getting new bucket")
			}

			return errors.Wrapf(
				bucket.Remove(ctx, c.String("name")),
				"problem removing %s from s3",
				c.String("name"),
			)
		},
	}
}

func s3DeletePrefixCmd() cli.Command {
	return cli.Command{
		Name:    "delete-prefix",
		Aliases: []string{"del-prefix", "rm-prefix"},
		Flags: baseS3Flags(
			cli.StringFlag{
				Name:  "prefix",
				Usage: "prefix of s3 key names",
			}),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			opts := pail.S3Options{
				SharedCredentialsProfile: c.String("profile"),
				Region:                   c.String("region"),
				Name:                     c.String("bucket"),
				DryRun:                   c.Bool("dry-run"),
				MaxRetries:               defaultMaxRetries,
			}
			bucket, err := pail.NewS3Bucket(opts)
			if err != nil {
				return errors.Wrap(err, "problem getting new bucket")
			}

			return errors.Wrapf(
				bucket.RemovePrefix(ctx, c.String("name")),
				"problem removing %s from s3",
				c.String("name"),
			)
		},
	}
}

func s3DeleteMatchingCmd() cli.Command {
	return cli.Command{
		Name:    "delete-match",
		Aliases: []string{"del-match", "rm-match"},
		Flags: baseS3Flags(
			cli.StringFlag{
				Name:  "match",
				Usage: "a regular expression definition",
			}),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			opts := pail.S3Options{
				SharedCredentialsProfile: c.String("profile"),
				Region:                   c.String("region"),
				Name:                     c.String("bucket"),
				DryRun:                   c.Bool("dry-run"),
				MaxRetries:               defaultMaxRetries,
			}
			bucket, err := pail.NewS3Bucket(opts)
			if err != nil {
				return errors.Wrap(err, "problem getting new bucket")
			}

			return errors.Wrapf(
				bucket.RemoveMatching(ctx, c.String("match")),
				"problem removing objects matching %s in s3",
				c.String("match"),
			)
		},
	}
}

func s3SyncToCmd() cli.Command {
	return cli.Command{
		Name:    "sync-to",
		Aliases: []string{"push"},
		Usage:   "sync changes from the local system to s3",
		Flags:   baseS3Flags(s3syncFlags()...),
		Action: func(c *cli.Context) error {
			ctx, cancel := ctxWithTimeout(c.Duration("timeout"))
			defer cancel()

			opts := pail.S3Options{
				SharedCredentialsProfile: c.String("profile"),
				Region:                   c.String("region"),
				Name:                     c.String("bucket"),
				DryRun:                   c.Bool("dry-run"),
				DeleteOnSync:             c.Bool("delete"),
				MaxRetries:               defaultMaxRetries,
				UseSingleFileChecksums:   true,
			}
			bucket, err := pail.NewS3Bucket(opts)
			if err != nil {
				return errors.Wrap(err, "problem getting new bucket")
			}
			if !c.Bool("serialize") {
				fmt.Println("HERE")
				syncOpts := pail.ParallelBucketOptions{
					Workers:      c.Int("workers"),
					DryRun:       c.Bool("dry-run"),
					DeleteOnSync: c.Bool("delete"),
				}
				bucket = pail.NewParallelSyncBucket(syncOpts, bucket)
			}

			return errors.Wrapf(
				bucket.Push(ctx, c.String("local"), c.String("prefix")),
				"problem syncing %s to s3",
				c.String("local"),
			)
		},
	}
}

func s3SyncFromCmd() cli.Command {
	return cli.Command{
		Name:    "sync-from",
		Aliases: []string{"pull"},
		Usage:   "sync changes from s3 to the local system",
		Flags:   baseS3Flags(s3syncFlags()...),
		Action: func(c *cli.Context) error {
			ctx, cancel := ctxWithTimeout(c.Duration("timeout"))
			defer cancel()

			opts := pail.S3Options{
				SharedCredentialsProfile: c.String("profile"),
				Region:                   c.String("region"),
				Name:                     c.String("bucket"),
				DryRun:                   c.Bool("dry-run"),
				DeleteOnSync:             c.Bool("delete"),
				MaxRetries:               defaultMaxRetries,
				UseSingleFileChecksums:   true,
			}
			bucket, err := pail.NewS3Bucket(opts)
			if err != nil {
				return errors.Wrap(err, "problem getting new bucket")
			}
			if !c.Bool("workers") {
				syncOpts := pail.ParallelBucketOptions{
					Workers:      c.Int("workers"),
					DryRun:       c.Bool("dry-run"),
					DeleteOnSync: c.Bool("delete"),
				}
				bucket = pail.NewParallelSyncBucket(syncOpts, bucket)
			}

			return errors.Wrapf(
				bucket.Pull(ctx, c.String("local"), c.String("prefix")),
				"problem syncing %s from  s3",
				c.String("prefix"),
			)
		},
	}
}

/////////////////////////////////////////////
//
// Implementations of Command Entry Points
//
/////////////////////////////////////////////

func ctxWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

/////////////////////////
//
// Option Generators
//
/////////////////////////

func baseS3Flags(args ...cli.Flag) []cli.Flag {
	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "region",
			Usage: "region to send requests to, defaults to us-east-1",
			Value: "us-east-1",
		},
		cli.StringFlag{
			Name:  "bucket",
			Usage: "the name of an s3 bucket",
		},
		cli.StringFlag{
			Name: "profile",
			Usage: fmt.Sprintln("set the AWS profile. By default reads from ENV vars and the default or",
				"AWS_PROFILE specified profile in ~/.aws/credentials."),
		},
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "make task operate in a dry-run mode",
		},
	}

	return append(flags, args...)
}

func s3syncFlags(args ...cli.Flag) []cli.Flag {
	pwd, _ := os.Getwd()

	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "local",
			Value: pwd,
			Usage: "a local path (directory)",
		},
		cli.StringFlag{
			Name:  "prefix",
			Usage: "a prefix of s3 key names",
		},
		cli.BoolFlag{
			Name:  "delete",
			Usage: "delete items from the target that do not exist in the source",
		},
		cli.DurationFlag{
			Name:  "timeout",
			Usage: "specify a timeout for operations, defaults to unlimited timeout if not specified",
		},
		cli.BoolFlag{
			Name:  "serialize",
			Usage: "serialize sync operation",
		},
		cli.IntFlag{
			Name:  "workers",
			Usage: "number of workers for parallelized sync operation, defaults to twice the number of logical CPUs",
			Value: 2 * runtime.NumCPU(),
		},
	}

	return append(flags, args...)
}

func s3opFlags(args ...cli.Flag) []cli.Flag {
	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "file",
			Usage: "a local path",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "the remote s3 resource name. may include the prefix.",
		},
	}

	return append(flags, args...)
}

func s3putFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "type",
			Usage: "standard MIME type describing the format of the data",
		},
		cli.StringFlag{
			Name:  "permissions",
			Usage: "canned ACL to apply to the file",
		},
	}
}
