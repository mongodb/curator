package greenbay

import (
	"encoding/json"
	"sync"

	"github.com/pkg/errors"
)

// Builder provides an interface to build a Configuration object
// programmatically. Primarily useful in testing or potentially for
// building out a greenbay-based test service.
type Builder struct {
	conf  Configuration
	mutex sync.RWMutex
}

// NewBuilder constructs a fully initialized Builder object.
func NewBuilder() *Builder {
	c := &Builder{
		conf: *newTestConfig(),
	}

	return c
}

// Conf returns a fully constructed and resolved Configuration,
// returning an error if there are any parsing or validation
// errors.
//
// Be aware that the config object returned is a *copy* of the config
// object in the builder, which allows you to generate and mutate
// generated multiple configs without impacting the builder's
// config. However, this operation does the correct but peculiar
// operation of copying a mutex in the config object.
//
// Additionally, in most cases you'll want a
// *pointer* to this object produced by this method.
func (b *Builder) Conf() (*Configuration, error) {
	out := &Configuration{}

	b.mutex.Lock()
	copy(out.RawTests, b.conf.RawTests)
	*out.Options = *b.conf.Options
	b.mutex.Unlock()

	out.reset()

	out.mutex.Lock()
	defer out.mutex.Unlock()
	if err := out.parseTests(); err != nil {
		return &Configuration{}, errors.Wrap(err, "refreshing config builder")
	}

	return out, nil
}

// AddCheck takes a greenbay.Checker implementation and adds it to the
// underlying config object.
func (b *Builder) AddCheck(check Checker) error {
	if check == nil {
		return errors.New("cannot add nil check")
	}

	// build the check structure
	t := rawTest{
		Name:      check.ID(),
		Suites:    check.Suites(),
		Operation: check.Name(),
	}

	raw, err := json.Marshal(check)
	if err != nil {
		return err
	}

	t.RawArgs = raw

	// take the lock and modify the structure.
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.conf.RawTests = append(b.conf.RawTests, t)

	return nil
}

// Len returns the number of checks represented in the builder config
// object.
func (b *Builder) Len() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return len(b.conf.RawTests)
}
