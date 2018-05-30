package greenbay

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	tempDir        string
	confFile       string
	numTestsInFile int
	conf           *Configuration
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

func (s *ConfigSuite) SetupSuite() {
	dir, err := ioutil.TempDir("", uuid.NewV4().String())
	s.Require().NoError(err)
	s.tempDir = dir

	conf := newTestConfig()
	s.numTestsInFile = 30

	jsonJob, err := json.Marshal(&mockShellCheck{
		shell:         job.NewShellJob("echo foo", ""),
		mockCheckBase: *newMockCheckBase("one", 0),
	})
	s.NoError(err)

	for i := 0; i < s.numTestsInFile; i++ {
		conf.RawTests = append(conf.RawTests,
			rawTest{
				Name:      fmt.Sprintf("check-working-shell-%d", i),
				Suites:    []string{"one", "two"},
				RawArgs:   jsonJob,
				Operation: mockShellCheckName,
			})
	}

	dump, err := json.Marshal(conf)
	s.Require().NoError(err)
	fn := filepath.Join(dir, "conf.json")
	s.confFile = fn
	err = ioutil.WriteFile(fn, dump, 0644)
	s.Require().NoError(err)
}

func (s *ConfigSuite) SetupTest() {
	s.conf = newTestConfig()
}

func (s *ConfigSuite) TearDownSuite() {
	s.Require().NoError(os.RemoveAll(s.tempDir))
}

func (s *ConfigSuite) TestTemporyFileConfigIsCorrect() {
	conf, err := ReadConfig(s.confFile)

	s.NoError(err)
	s.NotNil(conf)
}

func (s *ConfigSuite) TestInitializedConfObjectHasCorrectInitialValues() {
	s.NotNil(s.conf.tests)
	s.NotNil(s.conf.suites)

	s.Len(s.conf.tests, 0)
	s.Len(s.conf.suites, 0)
	s.Len(s.conf.RawTests, 0)

	s.Equal(runtime.NumCPU(), s.conf.Options.Jobs)
}

func (s *ConfigSuite) TestAddingDuplicateJobsToConfigDoesResultInDuplicateTests() {
	jsonJob, err := json.Marshal(&mockShellCheck{
		shell:         job.NewShellJob("echo foo", ""),
		mockCheckBase: *newMockCheckBase("foo", 0),
	})
	s.NoError(err)

	num := 3
	for i := 0; i < num; i++ {
		s.conf.RawTests = append(s.conf.RawTests,
			rawTest{
				Name:      "check-working-shell",
				Suites:    []string{"one", "two"},
				RawArgs:   jsonJob,
				Operation: mockShellCheckName,
			})
	}

	s.Len(s.conf.tests, 0)
	s.Error(s.conf.parseTests())

	s.Len(s.conf.tests, 1)
	s.Len(s.conf.suites, num-1)
	s.Len(s.conf.RawTests, num)
}

func (s *ConfigSuite) TestAddingInvalidDocumentsToConfig() {
	s.conf.RawTests = append(s.conf.RawTests,
		rawTest{
			Name:      "foo",
			Suites:    []string{"one", "two"},
			RawArgs:   []byte(`{a:1}`),
			Operation: "bar",
		})

	s.Len(s.conf.tests, 0)
	s.Len(s.conf.RawTests, 1)
	s.Error(s.conf.parseTests())
	s.Len(s.conf.tests, 0)
}

func (s *ConfigSuite) TestReadingConfigFromFileDoesntExist() {
	conf, err := ReadConfig(filepath.Join(s.tempDir, "foo", filepath.Base(s.confFile)))
	s.Error(err)
	s.Nil(conf)
}

func (s *ConfigSuite) TestReadConfigWithInvalidFormat() {
	fn := s.confFile + ".foo"
	err := os.Link(s.confFile, fn)
	s.NoError(err)

	conf, err := ReadConfig(fn)

	s.Error(err)
	s.Nil(conf)
}

func (s *ConfigSuite) TestForSuiteGetterObject() {
	conf, err := ReadConfig(s.confFile)

	s.NoError(err)
	s.NotNil(conf)

	tests := conf.TestsForSuites("one")

	c := 0
	for t := range tests {
		c++

		s.NoError(t.Err)
		s.NotNil(t.Job)
	}

	s.Equal(s.numTestsInFile, c)
}

func (s *ConfigSuite) TestForSuiteGetterGeneratorWithInvalidSuite() {
	tests := s.conf.TestsForSuites("DOES-NOT-EXIST", "ALSO-DOES-NOT-EXIST")
	s.NotNil(tests)
	for j := range tests {
		s.Error(j.Err)
		s.Nil(j.Job)
	}
}

func (s *ConfigSuite) TestByNameGenerator() {
	conf, err := ReadConfig(s.confFile)

	s.NoError(err)
	s.NotNil(conf)

	tests := conf.TestsByName("check-working-shell-1")

	c := 0
	for t := range tests {
		s.NoError(t.Err)
		s.NotNil(t.Job)
		c++
	}
	s.Equal(1, c)
}

func (s *ConfigSuite) TestByNameWithInvalidGenerator() {
	tests := s.conf.TestsByName("DOES-NOT-EXIST", "ALSO-DOES-NOT-EXIST")
	s.NotNil(tests)
	c := 0
	for j := range tests {
		if j.Err != nil {
			continue
		}

		c++
	}
	s.Equal(0, c)
}

func (s *ConfigSuite) TestsBySuiteDoesNotProduceDuplicates() {
	conf, err := ReadConfig(s.confFile)

	s.NoError(err)
	s.NotNil(conf)

	c := 0
	for t := range conf.TestsForSuites("one", "two") {
		// this could produce dupes because all the tests
		// appear in both suites
		s.NotNil(t)
		c++
	}

	s.Equal(s.numTestsInFile, c)
}

func (s *ConfigSuite) TestBySuiteWithInconsistentData() {
	conf, err := ReadConfig(s.confFile)

	s.NoError(err)
	s.NotNil(conf)

	// break the internal representation to make sure the
	// generator does the right thing.
	conf.tests = make(map[string]amboy.Job)

	tests := conf.TestsForSuites("one")

	for t := range tests {
		s.Error(t.Err)
		s.Nil(t.Job)
	}
}

func (s *ConfigSuite) TestCombinedCheckGenerator() {
	conf, err := ReadConfig(s.confFile)

	s.NoError(err)
	s.NotNil(conf)
	conf.tests = make(map[string]amboy.Job)

	s.Require().True(len(conf.RawTests) >= 2)
	conf.RawTests[0].Suites = []string{"foo"}
	conf.RawTests[1].Suites = []string{"bar"}

	var firstTestName string
	var tests int
	for range conf.GetAllTests([]string{}, []string{"foo", "bar"}) {
		tests++
	}
	s.Equal(2, tests)

	tests = 0
	for range conf.GetAllTests([]string{firstTestName}, []string{"one"}) {
		tests++
	}

	s.Equal(tests, s.numTestsInFile+1)
}
