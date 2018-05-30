package check

import (
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
)

func TestVersionComparison(t *testing.T) {
	assert := assert.New(t)

	smaller := semver.MustParse("1.0.0")
	larger := semver.MustParse("2.0.0")

	result, err := compareVersions("foo", smaller, larger)
	assert.Error(err)
	assert.False(result)

	result, err = compareVersions("gte", smaller, larger)
	assert.NoError(err)
	assert.False(result)

	result, err = compareVersions("gte", smaller, smaller)
	assert.NoError(err)
	assert.True(result)

	result, err = compareVersions("gte", larger, smaller)
	assert.NoError(err)
	assert.True(result)

	result, err = compareVersions("gt", smaller, larger)
	assert.NoError(err)
	assert.False(result)

	result, err = compareVersions("gt", larger, smaller)
	assert.NoError(err)
	assert.True(result)

	result, err = compareVersions("lt", smaller, larger)
	assert.NoError(err)
	assert.True(result)

	result, err = compareVersions("lt", larger, smaller)
	assert.NoError(err)
	assert.False(result)

	result, err = compareVersions("lte", larger, smaller)
	assert.NoError(err)
	assert.False(result)

	result, err = compareVersions("lte", smaller, larger)
	assert.NoError(err)
	assert.True(result)

	result, err = compareVersions("lte", larger, larger)
	assert.NoError(err)
	assert.True(result)

	result, err = compareVersions("eq", smaller, smaller)
	assert.NoError(err)
	assert.True(result)
}
