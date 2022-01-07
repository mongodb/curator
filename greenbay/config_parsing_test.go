package greenbay

import (
	"testing"

	"github.com/mongodb/grip"
	"github.com/stretchr/testify/assert"
)

func TestGetFormatFromFileName(t *testing.T) {
	assert := assert.New(t)

	// should return yaml
	for _, fn := range []string{"f.yaml", ".yaml", ".yml", "f.yml", ".json.yaml"} {
		frmt, err := getFormat(fn)
		assert.NoError(err)
		assert.Equal(formatYAML, frmt)
	}

	// should return json
	for _, fn := range []string{"f.json", ".json", ".yaml.abzckdfj_.json"} {
		frmt, err := getFormat(fn)
		assert.NoError(err)
		assert.Equal(formatJSON, frmt)
	}

	// should return error
	for _, fn := range []string{"json", "yaml", "f_json", "f_yaml", "foo.bson", "a.json-yaml", "b.yaml-json"} {
		frmt, err := getFormat(fn)
		assert.Error(err)
		assert.Equal(format(""), frmt)
	}

}

func TestGetJsonConfig(t *testing.T) {
	assert := assert.New(t) // nolint

	inputs := [][]byte{
		{},
		[]byte(`{foo: 1, bar: true}`),
		[]byte(`{}`),
		[]byte(`[]`),
	}

	// because all valid json is also valid yaml, we can sort of
	// fake this test, at least in the easy case:

	for _, data := range inputs {
		out, err := getJSONFormattedConfig(formatJSON, data)
		assert.Equal(out, data)
		assert.NoError(err)

		_, err = getJSONFormattedConfig(formatYAML, data)
		if !assert.NoError(err) {
			grip.Error(err)
		}
	}

	out, err := getJSONFormattedConfig(format("foo"), []byte{})
	assert.Error(err)
	assert.Nil(out)
}
