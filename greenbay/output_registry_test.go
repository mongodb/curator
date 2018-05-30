package greenbay

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestGlobalRegistry is a simple set of smoke tests for the
// results factory registry.
func TestGlobalRegistry(t *testing.T) {
	assert := assert.New(t)
	formats := []string{"gotest", "result"}

	// test private methods
	for _, name := range formats {
		factory, ok := resultRegistry.get(name)
		assert.True(ok)
		r := factory()
		assert.Implements((*ResultsProducer)(nil), r)
	}

	// same test, but with the public interface
	for _, name := range formats {
		factory, ok := GetResultsFactory(name)
		assert.True(ok)
		r := factory()
		assert.Implements((*ResultsProducer)(nil), r)
	}
}

type RegistrySuite struct {
	registry *resultsFactoryRegistry
	require  *require.Assertions
	suite.Suite
}

func TestRegistrySuite(t *testing.T) {
	suite.Run(t, new(RegistrySuite))
}

func (s *RegistrySuite) SetupSuite() {
	s.require = s.Require()
}

func (s *RegistrySuite) SetupTest() {
	s.registry = &resultsFactoryRegistry{
		factories: make(map[string]ResultsFactory),
	}
}

func (s *RegistrySuite) TestAddingDuplicateChecksIsNotAnError() {
	for i := 0; i < 20; i++ {
		s.registry.add("foo", func() ResultsProducer {
			return &GoTest{}
		})
		s.Len(s.registry.factories, 1)
	}
	_, ok := s.registry.get("foo")
	s.True(ok)
}

func (s *RegistrySuite) TestGettingNameThatDoesNotExistReturnsCorrectIndicator() {
	s.Len(s.registry.factories, 0)
	for i := 0; i < 20; i++ {
		f, ok := s.registry.get(fmt.Sprintf("output-%d", i))
		s.Nil(f)
		s.False(ok)
	}
}

func (s *RegistrySuite) TestRegistryProducesDifferentObjects() {
	// first set up registry instances
	s.registry.add("r", func() ResultsProducer {
		return &Results{}
	})

	s.registry.add("g", func() ResultsProducer {
		return &GoTest{}
	})

	s.Len(s.registry.factories, 2)

	f1, ok := s.registry.get("r")
	s.True(ok)
	f2, ok := s.registry.get("g")
	s.True(ok)
	s.NotEqual(fmt.Sprint(f1), fmt.Sprint(f2))
}
