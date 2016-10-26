package recall

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorAggregator(t *testing.T) {
	assert := assert.New(t)

	// hand a single channel with no errors, and observe no aggregate errors.
	e := make(chan error)
	close(e)
	assert.NoError(aggregateErrors(e))

	// hand a slice of channels with no errors and observe no aggregate error
	var chans []<-chan error
	for i := 0; i < 10; i++ {
		e = make(chan error)
		close(e)
		chans = append(chans, e)
	}

	assert.NoError(aggregateErrors(chans...))

	// now try it if the channels have errors in them.
	// first a single error.
	e = make(chan error, 1)
	e <- errors.New("foo")
	close(e)
	assert.Error(aggregateErrors(e))

	chans = chans[:0] // clear the slice
	// a slice of channels with one error
	for i := 0; i < 10; i++ {
		e = make(chan error, 2)
		e <- errors.New("foo")
		close(e)
		chans = append(chans, e)
	}

	assert.Error(aggregateErrors(chans...))

	// finally run a test with a lot of errors in a few channel
	chans = chans[:0]
	for i := 0; i < 5; i++ {
		e = make(chan error, 12)
		for i := 0; i < 10; i++ {
			e <- errors.New("foo")
		}

		close(e)
		chans = append(chans, e)
	}

	assert.Error(aggregateErrors(chans...))
}
