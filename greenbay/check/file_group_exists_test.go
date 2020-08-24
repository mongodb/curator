package check

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mongodb/amboy/registry"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FileGroupSuite struct {
	group   *fileGroup
	require *require.Assertions
	suite.Suite
}

func TestFileGroupSuite(t *testing.T) {
	suite.Run(t, new(FileGroupSuite))
}

func (s *FileGroupSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *FileGroupSuite) SetupTest() {
	factory, err := registry.GetJobFactory("file-group-all")
	s.require.NoError(err)
	check, ok := factory().(*fileGroup)
	s.require.True(ok)
	s.group = check
}

func (s *FileGroupSuite) TestWithInvalidRequirements() {
	s.group.Requirements.All = true
	s.group.Requirements.Any = true

	s.Error(s.group.Requirements.Validate())
	s.group.Run(context.Background())

	output := s.group.Output()
	s.True(output.Completed)
	s.False(output.Passed)
	s.Error(s.group.Error())
}

func (s *FileGroupSuite) TestOneExtantFileWithAllRequirement() {
	wd, err := os.Getwd()
	s.Require().NoError(err)
	wd = filepath.Dir(filepath.Dir(wd))
	s.group.FileNames = []string{filepath.Join(wd, "makefile")}
	s.True(s.group.Requirements.All)
	s.NoError(s.group.Requirements.Validate())

	s.group.Run(context.Background())
	output := s.group.Output()
	s.True(output.Completed)
	s.True(output.Passed)
	s.NoError(s.group.Error())
}

func (s *FileGroupSuite) TestWithFilesThatExistAndDoNotExistWIthAllRequirement() {
	wd, err := os.Getwd()
	s.Require().NoError(err)
	wd = filepath.Dir(filepath.Dir(wd))
	s.group.FileNames = []string{filepath.Join(wd, "makefile"), "makefile.DOES-NOT-EXIST"}
	s.True(s.group.Requirements.All)
	s.NoError(s.group.Requirements.Validate())

	s.group.Run(context.Background())
	output := s.group.Output()
	s.True(output.Completed)
	s.False(output.Passed)
	s.Error(s.group.Error())
}
