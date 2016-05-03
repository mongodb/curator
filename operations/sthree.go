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

For the sync jobs, you can use the aliases "push" and "pull", as in:

   curator s3 push --jobs <int> --bucket <bucket> --local <path> --prefix <remote>
   curator s3 pull --jobs <int> --bucket <bucket> --local <path> --prefix <remote>

For sync commands, the "jobs" argument is optional and defaults to 2
times the number of available processors. The "prefix" argument allows
you to sync only a portion of the bucket (e.g. all items with
key-names that start with that prefix.) For the "psuh" operation, the
curator prepends prefix to the local file name within the bucket.

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
	"os"
	"runtime"

	"github.com/codegangsta/cli"
	"github.com/mongodb/curator/sthree"
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
		Usage: "put a local file object into s3",
		Flags: s3opFlags(),
		Action: func(c *cli.Context) error {
			return s3Put(c.String("bucket"), c.String("file"), c.String("name"))
		},
	}
}

func s3GetCmd() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "download a local file object from s3",
		Flags: s3opFlags(),
		Action: func(c *cli.Context) error {
			return s3Get(c.String("bucket"), c.String("name"), c.String("file"))
		},
	}
}

func s3DeleteCmd() cli.Command {
	return cli.Command{
		Name:    "delete",
		Aliases: []string{"del", "rm"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "bucket",
				Usage: "the name of an s3 bucket",
			},
			cli.StringSliceFlag{
				Name:  "name",
				Usage: "the name of an object in s3",
			},
		},
		Action: func(c *cli.Context) error {
			return s3Delete(c.String("bucket"), c.StringSlice("name")...)
		},
	}
}

func s3DeletePrefixCmd() cli.Command {
	return cli.Command{
		Name:    "delete-prefix",
		Aliases: []string{"del-prefix", "rm-prefix"},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "bucket",
				Usage: "the name of an s3 bucket",
			},
			cli.StringFlag{
				Name:  "prefix",
				Usage: "a prefix of s3 key names",
			},
		},
		Action: func(c *cli.Context) error {
			return s3DeletePrefix(c.String("bucket"), c.String("prefix"))
		},
	}
}

func s3SyncToCmd() cli.Command {
	return cli.Command{
		Name:    "sync-to",
		Aliases: []string{"push"},
		Usage:   "sync changes from the local system to s3",
		Flags:   s3syncFlags(),
		Action: func(c *cli.Context) error {
			return s3SyncTo(c.String("bucket"), c.String("local"), c.String("prefix"), c.Int("jobs"))
		},
	}
}

func s3SyncFromCmd() cli.Command {
	return cli.Command{
		Name:    "sync-from",
		Aliases: []string{"pull"},
		Usage:   "sync changes from s3 to the local system",
		Flags:   s3syncFlags(),
		Action: func(c *cli.Context) error {
			return s3SyncFrom(c.String("bucket"), c.String("local"), c.String("prefix"), c.Int("jobs"))
		},
	}
}

/////////////////////////////////////////////
//
// Implementations of Command Entry Points
//
/////////////////////////////////////////////

// these helpers exist to facilitate easier unittesting

func s3Put(bucket, file, remoteFile string) error {
	b := sthree.GetBucket(bucket)

	err := b.Open()
	defer b.Close()

	if err != nil {
		return err
	}

	return b.Put(file, remoteFile)
}

func s3Get(bucket, remoteFile, file string) error {
	b := sthree.GetBucket(bucket)

	err := b.Open()
	defer b.Close()

	if err != nil {
		return err
	}

	return b.Get(remoteFile, file)
}

func s3Delete(bucket string, file ...string) error {
	b := sthree.GetBucket(bucket)

	err := b.Open()
	defer b.Close()
	if err != nil {
		return err
	}

	// DeleteMany handles the single-delete case gracefully, so
	// there's no use in adding complexity here.
	return b.DeleteMany(file...)
}

func s3DeletePrefix(bucket, prefix string) error {
	b := sthree.GetBucket(bucket)

	err := b.Open()
	defer b.Close()
	if err != nil {
		return err
	}

	return b.DeletePrefix(prefix)
}

func s3SyncTo(bucket, local, prefix string, jobs int) error {
	b := sthree.GetBucket(bucket)
	err := b.SetNumJobs(jobs)
	if err != nil {
		return err
	}

	err = b.Open()
	defer b.Close()

	if err != nil {
		return err
	}

	return b.SyncTo(local, prefix)
}

func s3SyncFrom(bucket, local, prefix string, jobs int) error {
	b := sthree.GetBucket(bucket)
	err := b.SetNumJobs(jobs)
	if err != nil {
		return err
	}

	err = b.Open()
	defer b.Close()
	if err != nil {
		return err
	}

	return b.SyncFrom(local, prefix)
}

/////////////////////////
//
// Option Generators
//
/////////////////////////

func s3syncFlags() []cli.Flag {
	pwd, _ := os.Getwd()

	return []cli.Flag{
		cli.StringFlag{
			Name:  "bucket",
			Usage: "the name of an s3 bucket",
		},
		cli.StringFlag{
			Name:  "local",
			Value: pwd,
			Usage: "a local path (directory)",
		},
		cli.StringFlag{
			Name:  "prefix",
			Usage: "a prefix of s3 key names",
		},
		cli.IntFlag{
			Name:  "jobs",
			Value: runtime.NumCPU() * 2,
			Usage: "number of parallel workers processing",
		},
	}
}

func s3opFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "bucket",
			Usage: "the name of an s3 bucket",
		},
		cli.StringFlag{
			Name:  "file",
			Usage: "a local path (directory)",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "the remote s3 resource name. may include the prefix.",
		},
	}
}
