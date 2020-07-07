package operations

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/glide/action"
	gpath "github.com/Masterminds/glide/path"
	"github.com/Masterminds/glide/repo"
	"github.com/mongodb/grip"
	"github.com/mongodb/jasper"
	"github.com/mongodb/jasper/options"
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

	_, err := os.Stat("makefile")
	evgMakefileExists := !os.IsNotExist(err)
	makefileFlag := cli.StringFlag{
		Name:  cleanCommandFlag,
		Usage: "command to run to perform clean up",
	}

	if evgMakefileExists {
		makefileFlag.Value = "make vendor-clean"
	}

	return cli.Command{
		Name: "revendor",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  joinFlagNames(packageFlagName, "p"),
				Usage: "the name of the package to upgrade",
			},
			cli.StringFlag{
				Name:  joinFlagNames(revisionFlagName, "r"),
				Usage: "the updated package revision",
			},
			makefileFlag,
		},
		Usage: "revendor an existing package in glide.lock",
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

			glidePath := filepath.Join(wd, gpath.LockFile)
			newGlideContent, err := updatedGlideFile(glidePath, pkg, rev)
			if err != nil {
				return errors.Wrapf(err, "error getting updated glide file content from %s", glidePath)
			}
			if err := ioutil.WriteFile(glidePath, []byte(newGlideContent), 0777); err != nil {
				return errors.Wrapf(err, "error writing glide file %s", glidePath)
			}

			vendorPath := filepath.Join(wd, gpath.VendorDir, pkg)
			if err := installDependencies(vendorPath); err != nil {
				return errors.Wrapf(err, "error installing dependencies")
			}

			return errors.Wrapf(cleanupDependencies(vendorPath, c.String(cleanCommandFlag)), "error cleaning up dependencies")
		},
	}
}

// updatedGlideFile returns the glide file content with the package version
// changed to the given revision.
func updatedGlideFile(path, pkg, rev string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrapf(err, "error reading file %s", path)
	}

	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		if strings.Contains(line, pkg) {
			if i+1 >= len(lines) {
				return "", errors.Errorf("no version specified for package %s", pkg)
			}
			// The whitespace has to be consistent with the current
			// amount of whitespace
			versionExpr := "^\\s+version:(.*)"
			versionRegex, err := regexp.Compile(versionExpr)
			if err != nil {
				return "", errors.Wrapf(err, "invalid regex %s given", versionExpr)
			}
			groups := versionRegex.FindStringSubmatch(lines[i+1])
			if groups == nil || len(groups) < 2 {
				return "", errors.Errorf("could not match version regex %s", versionRegex)
			}
			oldRev := groups[1]
			lines[i+1] = lines[i+1][:len(lines[i+1])-len(oldRev)]
			lines[i+1] = lines[i+1] + " " + rev
			found = true
			break
		}
	}
	if !found {
		return "", errors.Errorf("package %s not found in glide file %s", pkg, path)
	}

	return strings.Join(lines, "\n"), nil
}

func installDependencies(vendorPath string) error {
	if err := os.RemoveAll(vendorPath); err != nil {
		return errors.Wrapf(err, "error removing vendored package directory %s", vendorPath)
	}

	installer := repo.NewInstaller()
	action.EnsureGoVendor()
	// We can't strip the VCS in this call because of a bug in this
	// glide version that doesn't handle concurrent directory walking
	// and removal properly.
	action.Install(installer, false, false)

	return nil
}

func cleanupDependencies(vendorPath string, cleanCmd string) error {
	gitPath := filepath.Join(vendorPath, ".git")
	if err := os.RemoveAll(gitPath); err != nil {
		return errors.Wrapf(err, "could not remove package VCS directory %s", gitPath)
	}

	if cleanCmd == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	mgr, err := jasper.NewSynchronizedManager(false)
	if err != nil {
		return errors.Wrap(err, "problem creating process manager")
	}

	output := options.Output{Output: os.Stdout, SendErrorToOutput: true}
	return errors.Wrapf(mgr.CreateCommand(ctx).Append(cleanCmd).SetOutputOptions(output).Run(ctx), "could not run clean command %s", cleanCmd)
}
