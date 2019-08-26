package operations

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/Masterminds/glide/action"
	gpath "github.com/Masterminds/glide/path"
	"github.com/Masterminds/glide/repo"
	shlex "github.com/anmitsu/go-shlex"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Revendor returns a cli.Command that upgrades a vendored dependency using
// glide.
func Revendor() cli.Command {
	const (
		packageFlagName  = "package"
		revisionFlagName = "revision"
		cleanCommandFlag = "clean"
	)
	return cli.Command{
		Name: "revendor",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  packageFlagName,
				Usage: "the name of the package to upgrade",
			},
			cli.StringFlag{
				Name:  revisionFlagName,
				Usage: "the updated package revision",
			},
			cli.StringFlag{
				Name:  cleanCommandFlag,
				Usage: "command to run to perform clean up",
			},
		},
		Before: mergeBeforeFuncs(
			requireStringFlag(packageFlagName),
			requireStringFlag(revisionFlagName),
		),
		Action: func(c *cli.Context) error {
			pkg := c.String(packageFlagName)
			rev := c.String(revisionFlagName)
			if pkg == "" || rev == "" {
				grip.Info("missing package or revision, exiting")
				return nil
			}

			wd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "error getting working directory")
			}

			vendorPath := filepath.Join(wd, gpath.VendorDir, pkg)

			glidePath := filepath.Join(wd, gpath.LockFile)
			glideFile, err := os.Open(glidePath)
			if err != nil {
				return errors.Wrapf(err, "glide file %s could not be opened", glidePath)
			}
			glideContent, err := ioutil.ReadAll(glideFile)
			if err != nil {
				return errors.Wrapf(err, "error reading glide file %s", glidePath)
			}

			lines := strings.Split(string(glideContent), "\n")
			found := false
			for i, line := range lines {
				if strings.Contains(line, pkg) {
					if i+1 >= len(lines) {
						return errors.Wrapf(err, "no version specified for package %s", pkg)
					}
					// The whitespace has to be consistent with the current
					// amount of whitespace
					firstNonWhitespaceIdx := strings.IndexFunc(lines[i+1], func(r rune) bool {
						return !unicode.IsSpace(r)
					})
					if firstNonWhitespaceIdx == -1 {
						return errors.Wrapf(err, "missing version string on line following package")
					}
					whitespace := lines[i+1][:firstNonWhitespaceIdx]
					lines[i+1] = fmt.Sprintf("%sversion: %s", whitespace, rev)
					found = true
					break
				}
			}
			if !found {
				return errors.Errorf("package %s not found in glide file %s", pkg, glidePath)
			}

			if err := ioutil.WriteFile(glidePath, []byte(strings.Join(lines, "\n")), 0777); err != nil {
				return errors.Wrapf(err, "error writing glide file %s", glidePath)
			}

			if err := os.RemoveAll(vendorPath); err != nil {
				return errors.Wrapf(err, "error removing vendored package directory %s", vendorPath)
			}

			installer := repo.NewInstaller()
			action.EnsureGoVendor()
			// We can't strip the VCS in this call because of a bug in this
			// glide version that doesn't handle concurrent directory walking
			// and removal properly.
			action.Install(installer, false, false)

			gitPath := filepath.Join(vendorPath, ".git")
			if err := os.RemoveAll(gitPath); err != nil {
				return errors.Wrapf(err, "could not remove package VCS directory %s", gitPath)
			}

			cmdStr := c.String(cleanCommandFlag)
			if cmdStr == "" {
				return nil
			}

			splitCmd, err := shlex.Split(cmdStr, true)
			if err != nil {
				return errors.Wrapf(err, "could not parse clean command %s", cmdStr)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, splitCmd[0], splitCmd[1:]...)
			cmd.Stdout = os.Stdout
			return errors.Wrapf(cmd.Run(), "could not run clean command %s", cmdStr)
		},
	}
}
