package greenbay

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OptionsSuite struct {
	tmpDir  string
	opts    *OutputOptions
	require *require.Assertions
	queue   amboy.Queue
	cancel  context.CancelFunc
	suite.Suite
}

func TestOptionsSuite(t *testing.T) {
	suite.Run(t, new(OptionsSuite))
}

// Suite Fixtures:

func (s *OptionsSuite) SetupSuite() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.require = s.Require()

	tmpDir, err := ioutil.TempDir("", uuid.Must(uuid.NewV4()).String())
	s.require.NoError(err)
	s.tmpDir = tmpDir

	s.queue = queue.NewLocalUnordered(2)
	s.require.NoError(s.queue.Start(ctx))
	num := 5
	for i := 0; i < num; i++ {
		c := &mockCheck{}
		c.SetID(fmt.Sprintf("mock-check-%d", i))
		s.NoError(s.queue.Put(ctx, c))
	}
	s.Equal(num, s.queue.Stats(ctx).Total)
	amboy.Wait(ctx, s.queue)
}

func (s *OptionsSuite) SetupTest() {
	s.opts = &OutputOptions{}
}

func (s *OptionsSuite) TearDownSuite() {
	s.NoError(os.RemoveAll(s.tmpDir))
	s.cancel()
}

// Test cases:

func (s *OptionsSuite) TestConstructorDoesNotInvertsValueOfQuietArgument() {
	for _, q := range []bool{true, false} {
		opt, err := NewOutputOptions("", "gotest", q)
		s.NoError(err)
		s.Equal(q, opt.suppressPassing)
	}
}

func (s *OptionsSuite) TestEmptyFileNameDisablesWritingFiles() {
	opt, err := NewOutputOptions("", "gotest", true)
	s.NoError(err)
	s.Equal("", opt.fn)
	s.False(opt.writeFile)
}

func (s *OptionsSuite) TestSpecifiedFileEnablesWritingFiles() {
	fn := filepath.Join(s.tmpDir, "enabled-one")
	opt, err := NewOutputOptions(fn, "gotest", false)
	s.NoError(err)
	s.Equal(fn, opt.fn)
	s.True(opt.writeFile)
}

func (s *OptionsSuite) TestConstructorErrorsWithInvalidOutputFormats() {
	for _, format := range []string{"foo", "bar", "nothing", "NIL"} {
		opt, err := NewOutputOptions("", format, true)
		s.Error(err)
		s.Nil(opt)
	}
}

func (s *OptionsSuite) TestResultsProducderGeneratorErrorsWithInvalidFormat() {
	for _, format := range []string{"foo", "bar", "nothing", "NIL"} {
		s.opts.format = format
		rp, err := s.opts.GetResultsProducer()
		s.Error(err)
		s.Nil(rp)
	}
}

func (s *OptionsSuite) TestResultsProducerOperationFailsWIthInvaildFormat() {
	for _, format := range []string{"foo", "bar", "nothing", "NIL"} {
		s.opts.format = format
		err := s.opts.ProduceResults(context.TODO(), nil)
		s.Error(err)
	}
}

func (s *OptionsSuite) TestGetResultsProducerForValidFormats() {
	for _, format := range []string{"gotest", "result", "log"} {
		s.opts.format = format
		rp, err := s.opts.GetResultsProducer()
		s.NoError(err)
		s.NotNil(rp)
		s.Implements((*ResultsProducer)(nil), rp)
	}
}

func (s *OptionsSuite) TestResultsProducerOperationReturnsErrorWithNilQueue() {
	ctx := context.Background()
	for _, format := range []string{"gotest", "result", "log"} {
		opt, err := NewOutputOptions("", format, true)
		s.NoError(err)

		s.Error(opt.ProduceResults(ctx, nil))
	}
}

func (s *OptionsSuite) TestResultsToStandardOutButNotPrint() {
	for _, format := range []string{"gotest", "result", "log"} {
		opt, err := NewOutputOptions("", format, true)
		s.NoError(err)

		s.NoError(opt.ProduceResults(context.Background(), s.queue))
	}
}

func (s *OptionsSuite) TestResultsToFileOnly() {
	for idx, format := range []string{"gotest", "result", "log"} {
		fn := filepath.Join(s.tmpDir, fmt.Sprintf("enabled-two-%d", idx))
		opt, err := NewOutputOptions(fn, format, false)

		s.NoError(err)
		s.NoError(opt.ProduceResults(context.Background(), s.queue))
	}
}

func (s *OptionsSuite) TestResultsToFileAndOutput() {
	for idx, format := range []string{"gotest", "result", "log"} {
		fn := filepath.Join(s.tmpDir, fmt.Sprintf("enabled-three-%d", idx))
		opt, err := NewOutputOptions(fn, format, true)

		s.NoError(err)
		s.NoError(opt.ProduceResults(context.Background(), s.queue))
	}
}
