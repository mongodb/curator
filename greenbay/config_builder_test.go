package greenbay

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilderObject(t *testing.T) {
	assert := assert.New(t)
	builder := NewBuilder()

	assert.Equal(builder.Len(), 0)

	// TODO register type of check
	// assert.NoError(builder.AddCheck(j))
	// assert.Equal(builder.Len(), 1)
	// firstCopy, err := builder.Conf()
	// grip.Error(err)
	// assert.NoError(err)
	// assert.Equal(len(firstCopy.RawTests), 1)

	// assert.NoError(builder.AddCheck(j))
	// assert.Equal(builder.Len(), 2)
	// assert.Equal(len(firstCopy.RawTests), 1)

}
