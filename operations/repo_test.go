package operations

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mongodb/grip"
	"github.com/urfave/cli"
)

func (s *CommandsSuite) TestRepoFlags() {
	flags := repoFlags()

	names := make(map[string]bool)
	for _, flag := range flags {
		names[flag.GetName()] = true

		name := flag.GetName()
		if name == "dry-run" || name == "rebuild" {
			s.IsType(cli.BoolFlag{}, flag)
		} else if name == "timeout" {
			s.IsType(cli.DurationFlag{}, flag)
		} else if name == "retries" {
			s.IsType(cli.IntFlag{}, flag)
		} else {
			s.IsType(cli.StringFlag{}, flag)
		}
	}

	s.Len(names, 12)
	s.Len(flags, 12)
	s.True(names["config"])
	s.True(names["retries"])
	s.True(names["distro"])
	s.True(names["version"])
	s.True(names["edition"])
	s.True(names["timeout"])
	s.True(names["arch"])
	s.True(names["packages"])
	s.True(names["profile"])
	s.True(names["dry-run"])
}

func (s *CommandsSuite) TestRebuildOperationOnProcess() {
	err := os.Setenv("NOTARY_TOKEN", "foo")
	s.NoError(err)
	err = buildRepo(context.Background(),
		"./", // packages
		"repobuilder/config_test.yaml", // repo config path
		"build/repo-build-test",        // workingdir
		"rhel7",                        // distro
		"enterprise",                   // edition
		"2.8.0",                        // mongodbe version
		"x86_64",                       // arch
		"default",                      // aws profile
		true,                           // dryrun
		true,                           // rebuild
		1)                              // retries

	// TODO: we should be able to get a dry run that passes on
	// tests machines, but at the moment this depends on the
	// notary client and other configuration facts that aren't
	// necessarily true in the test environment.
	s.Error(err)
	grip.Warning(err)
}

func (s *CommandsSuite) TestDryRunOperationOnProcess() {
	err := buildRepo(context.Background(),
		"./", // packages
		"repobuilder/config_test.yaml", // repo config path
		"build/repo-build-test",        // workingdir
		"rhel7",                        // distro
		"enterprise",                   // edition
		"2.8.0",                        // mongodbe version
		"x86_64",                       // arch
		"default",                      // aws profile
		true,                           // dryrun
		false,                          // rebuild
		1)                              // retries

	s.Error(err)
	grip.Warning(err)
}

func (s *CommandsSuite) TestGetPackagesFunction() {
	cwd, err := filepath.Abs("../")
	s.NoError(err)

	testFiles, err := getPackages(cwd, "_test.go")
	s.NoError(err)
	for _, fn := range testFiles {
		s.True(filepath.IsAbs(fn))
		_, err = os.Stat(fn)
		s.False(os.IsNotExist(err))
	}

	goFiles, err := getPackages(cwd, ".go")
	s.NoError(err)
	for _, fn := range goFiles {
		s.True(filepath.IsAbs(fn))
		_, err = os.Stat(fn)
		s.False(os.IsNotExist(err))
	}

	noFiles, err := getPackages(cwd+".DOES_NOT_EXIST", "foo")
	s.Error(err)
	s.Len(noFiles, 0)
}
