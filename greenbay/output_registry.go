package greenbay

import (
	"bytes"
	"sync"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
)

// ResultsFactory defines the signature used by constructor functions
// for implementations of the ResultsProducer interface.
type ResultsFactory func() ResultsProducer

type resultsFactoryRegistry struct {
	factories map[string]ResultsFactory
	mutex     sync.RWMutex
}

var resultRegistry *resultsFactoryRegistry

func init() {
	resultRegistry = &resultsFactoryRegistry{
		factories: make(map[string]ResultsFactory),
	}

	AddFactory("gotest", func() ResultsProducer {
		return &GoTest{
			buf: bytes.NewBuffer([]byte{}),
		}
	})

	AddFactory("result", func() ResultsProducer {
		return &Results{}
	})

	AddFactory("log", func() ResultsProducer {
		return &GripOutput{}
	})

	AddFactory("json", func() ResultsProducer {
		return &JSONResults{}
	})

	AddFactory("report", func() ResultsProducer {
		return &Report{}
	})
}

func (r *resultsFactoryRegistry) add(name string, factory ResultsFactory) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, ok := r.factories[name]
	grip.AlertWhen(ok, message.Fields{
		"message": "overwriting existing factory",
		"factory": name,
	})

	r.factories[name] = factory
}

func (r *resultsFactoryRegistry) get(name string) (ResultsFactory, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	factory, ok := r.factories[name]

	grip.AlertWhen(!ok, message.Fields{
		"message": "factory does not exist",
		"name":    name,
	})

	return factory, ok
}

////////////////////////////////////////////////////////////////////////
//
// Public access methods for the global registry
//
////////////////////////////////////////////////////////////////////////

// GetResultsFactory provides a public mechanism for accessing
// constructors for result formats.
func GetResultsFactory(name string) (ResultsFactory, bool) {
	return resultRegistry.get(name)
}

// AddFactory provides a mechanism for adding additional results
// output to output registry.
func AddFactory(name string, factory ResultsFactory) {
	resultRegistry.add(name, factory)
}
