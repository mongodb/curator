package greenbay

import (
	"io/ioutil"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/mongodb/amboy"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// Helper functions that convert yaml-to-json so that the constructor
// can just convert to the required struct type.
type format string

const (
	formatYAML format = "yaml"
	formatJSON format = "json"
)

func getFormat(fn string) (format, error) {
	ext := filepath.Ext(fn)

	if ext == ".yaml" || ext == ".yml" {
		return formatYAML, nil
	} else if ext == ".json" {
		return formatJSON, nil
	}

	return "", errors.Errorf("greenbay does not support files with '%s' extension", ext)
}

func getJSONFormattedConfig(format format, data []byte) ([]byte, error) {
	var err error

	if format == formatYAML {
		data, err = yaml.YAMLToJSON(data)
		if err != nil {
			return nil, errors.Wrap(err, "problem parsing config")
		}

		return data, nil
	} else if format == formatJSON {
		return data, nil
	}

	return nil, errors.Errorf("%s is not a support format", format)
}

func getRawConfig(fn string) ([]byte, error) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, errors.Wrapf(err, "problem reading greenbay config file: %s", fn)
	}

	format, err := getFormat(fn)
	if err != nil {
		return nil, errors.Wrapf(err, "problem determining format of file %s", fn)
	}

	return getJSONFormattedConfig(format, data)
}

////////////////////////////////////////////////////////////////////////
//
// Internal Methods used by the constructor (ReadConfig) function.
//
////////////////////////////////////////////////////////////////////////

func (c *Configuration) parseTests() error {
	catcher := grip.NewCatcher()
	for _, msg := range c.RawTests {
		c.addSuites(msg.Name, msg.Suites)

		testJob, err := msg.resolveCheck()
		if err != nil {
			catcher.Add(errors.Wrapf(err, "problem resolving %s", msg.Name))
			continue
		}

		err = c.addTest(msg.Name, testJob)
		if err != nil {
			grip.Alert(err)
			catcher.Add(err)
			continue
		}

		grip.Debugln("added test named:", msg.Name, "type:", testJob.Name())
	}

	return catcher.Resolve()
}

// These methods are unsafe, and need to be used within the context a lock.

func (c *Configuration) addSuites(name string, suites []string) {
	for _, suite := range suites {
		if _, ok := c.suites[suite]; !ok {
			c.suites[suite] = []string{}
		}

		c.suites[suite] = append(c.suites[suite], name)
	}
}

func (c *Configuration) addTest(name string, j amboy.Job) error {
	if _, ok := c.tests[name]; ok {
		return errors.Errorf("two tests named '%s'", name)
	}

	c.tests[name] = j

	return nil
}

// end unsafe methods
