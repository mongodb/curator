/*
Configuration

The RepositoryConfig object provides some basic metadata used to
generate repositories in addition to information about every
repository.
*/
package repobuilder

import (
	"fmt"
	"io/ioutil"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// RepositoryConfig provides an interface and schema for the
// repository configuration file. These files contain some basic
// global configuration, and a list of repositories, controlled by the
// RepositoryDefinition type.
type RepositoryConfig struct {
	Repos    []*RepositoryDefinition `bson:"repos" json:"repos" yaml:"repos"`
	Services struct {
		NotaryURL string `bson:"notary_url" json:"notary_url" yaml:"notary_url"`
	} `bson:"services" json:"services" yaml:"services"`
	Templates struct {
		Index string            `bson:"index_page" json:"index_page" yaml:"index_page"`
		Deb   map[string]string `bson:"deb" json:"deb" yaml:"deb"`
	} `bson:"templates" json:"templates" yaml:"templates"`
	DryRun           bool   `bson:"dry_run" json:"dry_run" yaml:"dry_run"`
	Verbose          bool   `bson:"verbose" json:"verbose" yaml:"verbose"`
	WorkSpace        string `bson:"workspace" json:"workspace" yaml:"workspace"`
	TempSpace        string `bson:"temp" json:"temp" yaml:"temp"`
	Region           string `bson:"region" json:"region" yaml:"region"`
	fileName         string
	definitionLookup map[string]map[string]*RepositoryDefinition
}

// RepoType defines type of repositories.
type RepoType string

const (
	// RPM is a constant to refer to RPM repositories.
	RPM RepoType = "rpm"

	// DEB is a constant to refer to DEB repositories.
	DEB = "deb"
)

// RepositoryDefinition objects exist for each repository that we want to publish
type RepositoryDefinition struct {
	Name          string   `bson:"name" json:"name" yaml:"name"`
	Type          RepoType `bson:"type" json:"type" yaml:"type"`
	CodeName      string   `bson:"code_name" json:"code_name" yaml:"code_name"`
	Bucket        string   `bson:"bucket" json:"bucket" yaml:"bucket"`
	Region        string   `bson:"region" json:"region" yaml:"region"`
	Repos         []string `bson:"repos" json:"repos" yaml:"repos"`
	Edition       string   `bson:"edition" json:"edition" yaml:"edition"`
	Architectures []string `bson:"architectures,omitempty" json:"architectures,omitempty" yaml:"architectures,omitempty"`
	Component     string   `bson:"component" json:"component" yaml:"component"`
}

// NewRepositoryConfig produces a pointer to an initialized
// RepositoryConfig object.
func NewRepositoryConfig() *RepositoryConfig {
	c := &RepositoryConfig{
		definitionLookup: make(map[string]map[string]*RepositoryDefinition),
	}
	c.Templates.Deb = make(map[string]string)

	return c
}

// GetConfig takes the name of a file and returns a pointer to
// RepositoryConfig object. If the object is invalid or currupt in
// some way, the method returns a nil RepositoryConfig and an error.
func GetConfig(fileName string) (*RepositoryConfig, error) {
	c := NewRepositoryConfig()

	if err := c.read(fileName); err != nil {
		return nil, errors.WithStack(err)
	}

	if err := c.processRepos(); err != nil {
		return nil, errors.WithStack(err)
	}

	if c.Services.NotaryURL == "" {
		grip.Warning(message.Fields{
			"message":   "no notary service url specified",
			"file":      fileName,
			"num_repos": len(c.Repos),
		})
	}

	return c, nil
}

func (c *RepositoryConfig) read(fileName string) error {
	c.fileName = fileName

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return errors.Wrapf(err, "could not read file %v", fileName)
	}

	if err = yaml.Unmarshal(data, c); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(c.Validate())
}

// Validate ensures that the configuration file is correct, sets any
// unset defaults, and returns an error if there are any remaining
// errors.
func (c *RepositoryConfig) Validate() error {
	if c.Region == "" {
		c.Region = "us-east-1"
	}

	return nil
}

func (c *RepositoryConfig) processRepos() error {
	catcher := grip.NewCatcher()

	for idx, dfn := range c.Repos {
		// do some basic validation that the type value is correct.
		if dfn.Type != DEB && dfn.Type != RPM {
			catcher.Add(fmt.Errorf("%s is not a valid repo type", dfn.Type))
		}

		// build the definitionLookup map
		if _, ok := c.definitionLookup[dfn.Edition]; !ok {
			c.definitionLookup[dfn.Edition] = make(map[string]*RepositoryDefinition)
		}

		// this lets us detect if there are duplicate
		// repository/edition pairs.
		if _, ok := c.definitionLookup[dfn.Edition][dfn.Name]; ok {
			catcher.Add(fmt.Errorf("the %s.%s already exists as repo #%d",
				dfn.Edition, dfn.Name, idx))
			continue
		}

		if dfn.Type == DEB && len(dfn.Architectures) == 0 {
			catcher.Add(fmt.Errorf("debian distro %s does not specify architecture list",
				dfn.Name))
			continue
		}

		if dfn.Region == "" {
			dfn.Region = c.Region
		}

		c.definitionLookup[dfn.Edition][dfn.Name] = dfn
	}

	return catcher.Resolve()
}

// GetRepositoryDefinition takes the name of as repository and an edition,
// return a repository configuration. The second value is true when
// the requested edition+name exists, and false otherwise. When the
// requested edition+name does not exist, the value is nil.
func (c *RepositoryConfig) GetRepositoryDefinition(name, edition string) (*RepositoryDefinition, bool) {
	e, ok := c.definitionLookup[edition]
	if !ok {
		return nil, false
	}

	dfn, ok := e[name]
	if !ok {
		return nil, false
	}

	return dfn, true
}

func (c *RepositoryDefinition) getArchForDistro(arch string) string {
	if c.Type == DEB {
		if arch == "x86_64" {
			return "amd64"
		} else if arch == "ppc64le" {
			return "ppc64el"
		}
	}

	return arch
}
