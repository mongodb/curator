package repobuilder

import (
	"github.com/evergreen-ci/bond"
	"github.com/mongodb/grip"
)

// JobOptions describes the options to run a job that builds a repo.
type JobOptions struct {
	Configuration *RepositoryConfig     `bson:"conf" json:"conf" yaml:"conf"`
	Distro        *RepositoryDefinition `bson:"distro" json:"distro" yaml:"distro"`
	Version       string                `bson:"version" json:"version" yaml:"version"`
	Arch          string                `bson:"arch" json:"arch" yaml:"arch"`
	Packages      []string              `bson:"packages" json:"packages" yaml:"packages"`
	JobID         string                `bson:"job_id" json:"job_id" yaml:"job_id"`

	AWSProfile string `bson:"aws_profile" json:"aws_profile" yaml:"aws_profile"`
	AWSKey     string `bson:"aws_key" json:"aws_key" yaml:"aws_key"`
	AWSSecret  string `bson:"aws_secret" json:"aws_secret" yaml:"aws_secret"`
	AWSToken   string `bson:"aws_token" json:"aws_token" yaml:"aws_token"`

	NotaryKey   string `bson:"notary_key" json:"notary_key" yaml:"notary_key"`
	NotaryToken string `bson:"notary_token" json:"notary_token" yaml:"notary_token"`

	release bond.MongoDBVersion
}

// Validate returns an error if the job options struct is not logically valid.
func (opts *JobOptions) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(opts.Configuration == nil, "configuration must not be nil")
	catcher.NewWhen(opts.Distro == nil, "distro specification must not be nil")

	release, err := bond.CreateMongoDBVersion(opts.Version)
	catcher.Add(err)
	opts.release = release

	return catcher.Resolve()
}
