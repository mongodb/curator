package repobuilder

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/evergreen-ci/bond"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/grip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RepoJobSuite struct {
	j       *repoBuilderJob
	require *require.Assertions
	suite.Suite
}

func TestRepoJobSuite(t *testing.T) {
	suite.Run(t, new(RepoJobSuite))
}

func (s *RepoJobSuite) SetupSuite() {
	s.require = s.Require()
}

func (s *RepoJobSuite) SetupTest() {
	s.j = buildRepoJob()
	s.require.NotNil(s.j)
}

func (s *RepoJobSuite) TearDownTest() {
	for _, path := range s.j.workingDirs {
		grip.Error(os.RemoveAll(filepath.Join(s.j.Conf.WorkSpace, path)))
	}
}

func (s *RepoJobSuite) TestDependencyAccessorIsCorrect() {
	s.Equal("always", s.j.Dependency().Type().Name)
}

func (s *RepoJobSuite) TestSetDependencyAcceptsDifferentAlwaysRunInstances() {
	originalDep := s.j.Dependency()
	newDep := dependency.NewAlways()
	s.True(originalDep != newDep)

	s.j.SetDependency(newDep)
	s.True(originalDep != s.j.Dependency())
	s.Exactly(newDep, s.j.Dependency())
}

func TestProcessPackages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	j := buildRepoJob()
	var err error
	j.release, err = bond.NewMongoDBVersion("4.2.5-rc1")
	require.NoError(t, err)
	j.client = utility.GetDefaultHTTPRetryableClient()
	j.Distro = &RepositoryDefinition{Name: "test"}
	defer func() { utility.PutHTTPClient(j.client) }()
	j.PackagePaths = []string{"https://s3.amazonaws.com/mciuploads/mongo-release/enterprise-rhel-62-64-bit/98d10b50208db52f3aa0f16a634ec6fa73d465bc/artifacts/mongo_release_enterprise_rhel_62_64_bit_98d10b50208db52f3aa0f16a634ec6fa73d465bc_20_03_19_17_13_06-packages.tgz"}
	j.tmpdir, err = ioutil.TempDir("", "test-process-packages")
	require.NoError(t, err)

	assert.NoError(t, j.processPackages(ctx))
	assert.Len(t, j.PackagePaths, 6)
}
