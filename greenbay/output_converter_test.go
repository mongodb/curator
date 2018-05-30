package greenbay

import (
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/stretchr/testify/assert"
)

func TestConverter(t *testing.T) {
	assert := assert.New(t)

	j := job.NewShellJob("echo foo", "")
	assert.NotNil(j)
	c, err := convert(j)
	assert.Error(err)
	assert.Nil(c)

	mc := &mockCheck{}
	assert.Implements((*amboy.Job)(nil), mc)
	assert.Implements((*Checker)(nil), mc)

	c, err = convert(mc)
	assert.NoError(err)
	assert.NotNil(c)
}

func TestJobToCheckGenerator(t *testing.T) {
	assert := assert.New(t) // nolint
	input := make(chan amboy.Job)
	output := jobsToCheck(false, input)

	i := &mockCheck{}
	assert.Implements((*amboy.Job)(nil), i)
	input <- i

	o := <-output
	assert.NoError(o.err)
	assert.IsType(CheckOutput{}, o.output)

	close(input)
}
