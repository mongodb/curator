package repobuilder

import "context"

func (s *RepoJobSuite) TestRPMConstructedObjectHasExpectedValues() {
	conf, err := GetConfig("repobuilder/config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("rhel5", "enterprise")
	s.True(ok)
	s.createJob(conf, repo, "2.8.8", "x86_64", "default", "foo", "bar", "baz")

	// basic checks to make sure we create the instance
	s.Equal(s.j.Version, s.j.release.String())
	s.Equal([]string{"foo", "bar", "baz"}, s.j.PackagePaths)
	s.False(s.j.Conf.DryRun)
}

func (s *RepoJobSuite) TestRPMConstructorReturnsErrorForInvalidVersion() {
	conf, err := GetConfig("repobuilder/config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("rhel5", "enterprise")
	s.True(ok)
	_, err = NewBuildRepoJob(conf, repo, "2.8.8.8", "x86_64", "default", "foo", "bar", "baz")

	s.Error(err)
}

func (s *RepoJobSuite) TestRPMCompletedSetter() {
	conf, err := GetConfig("repobuilder/config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("rhel5", "enterprise")
	repo.Bucket = "build-curator-testing"
	s.True(ok)
	s.createJob(conf, repo, "2.8.8", "x86_64", "default")

	s.False(s.j.Conf.DryRun)
	s.j.Conf.DryRun = true

	s.False(s.j.Status().Completed)

	s.j.Run(context.Background())
	s.True(s.j.Status().Completed)
}
