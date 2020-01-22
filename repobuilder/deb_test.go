package repobuilder

import "context"

func (s *RepoJobSuite) createJob(conf *RepositoryConfig, distro *RepositoryDefinition, version, arch, profile string, pkgs ...string) {
	j, err := NewBuildRepoJob(conf, distro, version, arch, profile, pkgs...)
	s.require.NoError(err)
	var ok bool
	s.j, ok = j.(*repoBuilderJob)
	s.require.True(ok)
}

func (s *RepoJobSuite) TestDEBConstructedObjectHasExpectedValues() {
	conf, err := GetConfig("repobuilder/config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("debian8", "enterprise")
	s.True(ok)
	s.createJob(conf, repo, "2.8.8", "x86_64", "default", "foo", "bar", "baz")

	// basic checks to make sure we create the instance
	s.Equal(s.j.Version, s.j.release.String())
	s.Equal([]string{"foo", "bar", "baz"}, s.j.PackagePaths)
	s.False(s.j.Conf.DryRun)
}

func (s *RepoJobSuite) TestDEBConstructorReturnsErrorForInvalidVersion() {
	conf, err := GetConfig("repobuilder/config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("debian8", "enterprise")
	s.True(ok)
	_, err = NewBuildRepoJob(conf, repo, "2.8.8.8", "x86_64", "default", "foo", "bar", "baz")
	s.Error(err)
}

func (s *RepoJobSuite) TestDEBCompletedSetter() {
	conf, err := GetConfig("repobuilder/config_test.yaml")
	s.NoError(err)
	repo, ok := conf.GetRepositoryDefinition("debian8", "enterprise")
	repo.Bucket = "build-curator-testing"
	s.True(ok)
	s.createJob(conf, repo, "2.8.8", "x86_64", "default")

	s.False(s.j.Conf.DryRun)
	s.j.Conf.DryRun = true

	s.False(s.j.Status().Completed)

	s.j.Run(context.Background())
	s.True(s.j.Status().Completed)
}
