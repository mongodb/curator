package operations

import (
	"fmt"
	"os"
	"path/filepath"

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
		} else {
			s.IsType(cli.StringFlag{}, flag)
		}
	}

	s.Len(names, 10)
	s.Len(flags, 10)
	s.True(names["config"])
	s.True(names["distro"])
	s.True(names["version"])
	s.True(names["edition"])
	s.True(names["arch"])
	s.True(names["packages"])
	s.True(names["profile"])
	s.True(names["dry-run"])
}

func (s *CommandsSuite) TestRebuildOperationOnProcess() {
	os.Setenv("NOTARY_TOKEN", "foo")
	err := buildRepo(
		"./", // packages
		"../repobuilder/config_test.yaml", // repo config path
		"../build/repo-build-test",        // workingdir
		"rhel7",                           // distro
		"enterprise",                      // edition
		"2.8.0",                           // mongodbe version
		"x86_64",                          // arch
		"default",                         // aws profile
		true,                              // dryrun
		true)                              // rebuild

	if !s.NoError(err) {
		fmt.Println(err)
	}
}

func (s *CommandsSuite) TestDryRunOperationOnProcess() {
	err := buildRepo(
		"./", // packages
		"../repobuilder/config_test.yaml", // repo config path
		"../build/repo-build-test",        // workingdir
		"rhel7",                           // distro
		"enterprise",                      // edition
		"2.8.0",                           // mongodbe version
		"x86_64",                          // arch
		"default",                         // aws profile
		true,                              // dryrun
		false)                             // rebuild

	if !s.Equal(err.Error(), "no packages found in path './'") {
		fmt.Println(err)
	}
}

func (s *CommandsSuite) TestGetPackagesFunction() {
	cwd, err := filepath.Abs("../")
	s.NoError(err)

	testFiles, err := getPackages(cwd, "_test.go")
	s.NoError(err)
	for _, fn := range testFiles {
		s.True(filepath.IsAbs(fn))
		_, err := os.Stat(fn)
		s.False(os.IsNotExist(err))
	}

	goFiles, err := getPackages(cwd, ".go")
	s.NoError(err)
	for _, fn := range goFiles {
		s.True(filepath.IsAbs(fn))
		_, err := os.Stat(fn)
		s.False(os.IsNotExist(err))
	}

	noFiles, err := getPackages(cwd+".DOES_NOT_EXIST", "foo")
	s.Error(err)
	s.Len(noFiles, 0)
}
