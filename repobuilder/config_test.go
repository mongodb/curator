package repobuilder

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RepoConfigSuite struct {
	conf          *RepositoryConfig
	file          string
	invalidFile   string
	incorrectFile string
	require       *require.Assertions
	suite.Suite
}

func TestRepoConfigSuite(t *testing.T) {
	suite.Run(t, new(RepoConfigSuite))
}

func (s *RepoConfigSuite) SetupSuite() {
	s.require = s.Require()

	fn, err := filepath.Abs("config_test.yaml")
	s.require.NoError(err)
	s.file = fn

	invalidFn, err := filepath.Abs("config_invalid_test.yaml")
	s.require.NoError(err)
	s.invalidFile = invalidFn

	incorrectFn, err := filepath.Abs("config_incorrect_test.yaml")
	s.require.NoError(err)
	s.incorrectFile = incorrectFn
}

func (s *RepoConfigSuite) SetupTest() {
	s.conf = NewRepositoryConfig()
}

func (s *RepoConfigSuite) TestExampleIsReadableAndProducesNoErrors() {
	err := s.conf.read(s.file)
	s.NoError(err)
}

func (s *RepoConfigSuite) TestFilesThatDoNotExistProduceError() {
	err := s.conf.read(s.file + "-DOES-NOT-EXIST")
	s.Error(err)
}

func (s *RepoConfigSuite) TestExampleConfigHasNoInternalErrors() {
	err := s.conf.read(s.file)
	s.NoError(err)

	err = s.conf.processRepos()
	s.NoError(err)
}

func (s *RepoConfigSuite) TestConfigLoadFunctionReturnsObjectWithNoError() {
	conf, err := GetConfig(s.file)
	s.Require().NoError(err)
	s.IsType(s.conf, conf)
}

func (s *RepoConfigSuite) TestConfigLoadFunctionReturnsErrorIfFileDoesNotExist() {
	conf, err := GetConfig(s.file + "-DOES-NOT-EXIST")
	s.Error(err)
	s.Nil(conf)
}

func (s *RepoConfigSuite) TestInvalidConfigProcessReturnsError() {
	conf, err := GetConfig(s.invalidFile)
	s.Error(err)
	s.Nil(conf)
}

func (s *RepoConfigSuite) TestInvalidConfigErrorsAtReadStage() {
	err := s.conf.read(s.invalidFile)
	s.Error(err)
}

func (s *RepoConfigSuite) TestIncorrectConfigProcessReturnsError() {
	conf, err := GetConfig(s.incorrectFile)
	s.Error(err)
	s.Nil(conf)
}

func (s *RepoConfigSuite) TestGetRepoMethodReturnsNilObjectsForInvalidDefinitions() {
	var err error
	s.conf, err = GetConfig(s.file)
	s.Require().NoError(err)

	repo, ok := s.conf.GetRepositoryDefinition("rhel5", "subscription")
	s.False(ok)
	s.Nil(repo)
}

func (s *RepoConfigSuite) TestGetRepoMethodReturnsNilObjectsForInvalidName() {
	var err error

	s.conf, err = GetConfig(s.file)
	s.Require().NoError(err)

	repo, ok := s.conf.GetRepositoryDefinition("rhel55", "org")
	s.False(ok)
	s.Nil(repo)
}

func (s *RepoConfigSuite) TestGetRepoMethodReturnsExpectedRepoObject() {
	var err error

	s.conf, err = GetConfig(s.file)
	s.Require().NoError(err)

	rhelCommunity, ok := s.conf.GetRepositoryDefinition("rhel7", "org")
	s.require.True(ok)
	s.Equal("rhel7", rhelCommunity.Name)
	s.Equal("org", rhelCommunity.Edition)
	s.Equal("repo-test.mongodb.org", rhelCommunity.Bucket)
	s.Equal(RPM, rhelCommunity.Type)
	s.Len(rhelCommunity.Repos, 2)

	rhelEnterprise, ok := s.conf.GetRepositoryDefinition("rhel7", "enterprise")
	s.require.True(ok)
	s.NotEqual(rhelCommunity, rhelEnterprise)

	s.Equal("rhel7", rhelEnterprise.Name)
	s.Equal("enterprise", rhelEnterprise.Edition)
	s.Equal("repo-test.mongodb.com", rhelEnterprise.Bucket)
	s.Equal(RPM, rhelEnterprise.Type)
	s.Len(rhelEnterprise.Repos, 2)
}
