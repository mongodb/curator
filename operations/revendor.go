package operations

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/glide/action"
	gpath "github.com/Masterminds/glide/path"
	"github.com/Masterminds/glide/repo"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func Revendor() cli.Command {
	const (
		packageFlagName  = "package"
		revisionFlagName = "revision"
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
		},
		Action: func(c *cli.Context) error {
			pkg := c.String(packageFlagName)
			rev := c.String(revisionFlagName)
			if pkg == "" || rev == "" {
				grip.Info("missing package or revision, exiting")
				return nil
			}

			wd, err := os.Getwd()
			grip.EmergencyFatal(errors.Wrap(err, "error getting working directory"))

			vendorPath := filepath.Join(wd, gpath.VendorDir, pkg)

			glidePath := filepath.Join(wd, gpath.LockFile)
			_, err = os.Stat(glidePath)
			grip.EmergencyFatal(errors.Wrapf(err, "glide file %s not found", glidePath))

			glideFile, err := os.Open(glidePath)
			grip.EmergencyFatal(errors.Wrap(err, "error opening glide file"))
			glide, err := ioutil.ReadAll(glideFile)
			grip.EmergencyFatal(errors.Wrap(err, "error reading glide file"))

			lines := strings.Split(string(glide), "\n")
			found := false
			for i, line := range lines {
				if strings.Contains(line, pkg) {
					lines[i+1] = fmt.Sprintf("  version: %s", rev)
					found = true
					break
				}
			}
			if !found {
				grip.EmergencyFatalf("package %s not found in glide file", pkg)
			}

			grip.EmergencyFatal(errors.Wrap(ioutil.WriteFile(glidePath, []byte(strings.Join(lines, "\n")), 0777), "error writing glide file"))

			grip.EmergencyFatal(errors.Wrap(os.RemoveAll(vendorPath), "error removing vendor directory"))

			installer := repo.NewInstaller()
			action.EnsureGoVendor()
			action.Install(installer, false, false)

			stat, err := os.Stat(vendorPath)
			grip.EmergencyFatal(errors.Wrapf(err, "vendor directory %s not found", vendorPath))
			if !stat.IsDir() {
				grip.EmergencyFatalf("'%s' is not a directory", vendorPath)
			}
			gitPath := filepath.Join(vendorPath, ".git")
			grip.EmergencyFatal(errors.Wrap(os.RemoveAll(gitPath), "error removing .git directory"))
			return nil
		},
	}
}
