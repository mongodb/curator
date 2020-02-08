package operations

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Archive provides a top level command for interacting with tarball
// archives in a cross-platform manner.
func Archive() cli.Command {
	return cli.Command{
		Name: "archive",
		Subcommands: []cli.Command{
			MakeTarball(),
		},
	}
}

// MakeTarball produces a tarball.
func MakeTarball() cli.Command {
	return cli.Command{
		Name: "create",
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "item",
				Usage: "specify items to add to the archive",
			},
			cli.StringSliceFlag{
				Name:  "exclude",
				Usage: "regular expressions to exclude files",
			},
			cli.StringFlag{
				Name:  "prefix",
				Usage: "prefix of path within archive",
			},
			cli.StringFlag{
				Name:  "name",
				Value: "archive.tar.gz",
				Usage: "specify the name of the archive to create",
			},
		},
		Action: func(c *cli.Context) error {
			return errors.WithStack(createArchive(c.String("name"), c.String("prefix"), c.StringSlice("item"), c.StringSlice("exclude")))
		},
	}
}

// inspired by https://gist.github.com/jonmorehouse/9060515

type archiveWorkUnit struct {
	path string
	stat os.FileInfo
}

func getContents(paths []string, exclusions []string) <-chan archiveWorkUnit {
	var matchers []*regexp.Regexp
	for _, pattern := range exclusions {
		matchers = append(matchers, regexp.MustCompile(pattern))
	}

	output := make(chan archiveWorkUnit)

	go func() {
		for _, path := range paths {
			err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return errors.WithStack(err)
				}

				if info.IsDir() {
					return nil
				}

				for _, exclude := range matchers {
					if exclude.MatchString(p) {
						return nil
					}
				}
				output <- archiveWorkUnit{
					path: p,
					stat: info,
				}
				return nil
			})

			if err != nil {
				panic(errors.Wrap(err, "caught error walking file system"))
			}
		}
		close(output)
	}()

	return output
}

func addFile(tw *tar.Writer, prefix string, unit archiveWorkUnit) error {
	fn, err := filepath.EvalSymlinks(unit.path)
	if err != nil {
		return err
	}

	file, err := os.Open(fn)
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() { grip.Error(file.Close()) }()
	// now lets create the header as needed for this file within the tarball
	header := new(tar.Header)
	header.Name = filepath.Join(prefix, unit.path)
	header.Size = unit.stat.Size()
	header.Mode = int64(unit.stat.Mode())
	header.ModTime = unit.stat.ModTime()
	// write the header to the tarball archive
	if err := tw.WriteHeader(header); err != nil {
		return errors.WithStack(err)
	}
	// copy the file data to the tarball
	if _, err := io.Copy(tw, file); err != nil {
		return errors.WithStack(err)
	}

	grip.Debug(message.Fields{
		"message": "added file to archive",
		"name":    header.Name,
	})

	return nil
}

func createArchive(fileName, prefix string, paths []string, exclude []string) error {
	// set up the output file
	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "problem creating file %s", fileName)
	}
	defer func() { grip.Error(errors.Wrapf(file.Close(), "problem closing file %s", fileName)) }()

	// set up the  gzip writer
	gw := gzip.NewWriter(file)
	defer func() { grip.Error(errors.Wrapf(gw.Close(), "problem closing gzip writer %s", fileName)) }()
	tw := tar.NewWriter(gw)
	defer func() { grip.Error(errors.Wrapf(tw.Close(), "problem closing tar writer %s", fileName)) }()

	grip.Infoln("creating archive:", fileName)

	for unit := range getContents(paths, exclude) {
		err := addFile(tw, prefix, unit)
		if err != nil {
			return errors.Wrapf(err, "error adding path: %s [%+v]",
				unit.path, unit)
		}
	}

	return nil
}
