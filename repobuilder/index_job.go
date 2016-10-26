package repobuilder

import (
	"github.com/goamz/goamz/s3"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/curator/sthree"
	"github.com/pkg/errors"
	"github.com/tychoish/grip"
)

// IndexBuildJob implements the amboy.Job interface and provides a
// mechanism to *only* rebuild index pages for a repository.
type IndexBuildJob struct {
	Conf      *RepositoryConfig `bson:"conf" json:"conf" yaml:"conf"`
	Bucket    string            `bson:"bucket" json:"bucket" yaml:"bucket"`
	Profile   string            `bson:"aws_profile" json:"aws_profile" yaml:"aws_profile"`
	WorkSpace string            `bson:"local_workdir" json:"local_workdir" yaml:"local_workdir"`
	RepoName  string            `bson:"repo_name" json:"repo_name" yaml:"repo_name"`
	DryRun    bool              `bson:"dry_run" json:"dry_run" yaml:"dry_run"`

	*JobBase `bson:"base" json:"base" yaml:"base"`
}

func init() {
	registry.AddJobType("build-index-pages", func() amboy.Job {
		return &IndexBuildJob{}
	})
}

// NewIndexBuildJob constructs an IndexBuildJob object.
func NewIndexBuildJob(conf *RepositoryConfig, workSpace, repoName, bucket string, dryRun bool) *IndexBuildJob {
	j := &IndexBuildJob{
		Conf:      conf,
		DryRun:    dryRun,
		WorkSpace: workSpace,
		Bucket:    bucket,
		RepoName:  repoName,
		JobBase: &JobBase{
			JobType: amboy.JobType{
				Name:    "build-index-pages",
				Version: 0,
			},
		},
	}

	j.SetDependency(dependency.NewAlways())

	return j
}

// Run downloads the repository, and generates index pages at all
// levels of the repo.
func (j *IndexBuildJob) Run() {
	bucket := sthree.GetBucketWithProfile(j.Bucket, j.Profile)

	err := bucket.Open()
	if err != nil {
		j.addError(errors.Wrapf(err, "opening bucket %s", bucket))
		return
	}
	defer bucket.Close()

	if j.DryRun {
		// the error (second argument) will be caught (when we
		// run open below)
		bucket, err = bucket.DryRunClone()
		if err != nil {
			j.addError(errors.Wrapf(err,
				"problem getting bucket '%s' in dry-mode", bucket))
			return
		}

		err = bucket.Open()
		if err != nil {
			j.addError(errors.Wrapf(err, "opening bucket %s [dry-run]", bucket))
			return
		}
		defer bucket.Close()
	}
	bucket.NewFilePermission = s3.PublicRead

	defer j.markComplete()

	grip.Infof("downloading from %s to %s", bucket, j.WorkSpace)
	err = bucket.SyncFrom(j.WorkSpace, "", false)
	if err != nil {
		j.addError(errors.Wrapf(err, "sync from %s to %s", bucket, j.WorkSpace))
		return
	}

	if j.RepoName == "" {
		j.RepoName = j.Bucket
	}

	err = j.Conf.BuildIndexPageForDirectory(j.WorkSpace, j.RepoName)
	if err != nil {
		j.addError(errors.Wrapf(err, "building index.html pages for %s", j.WorkSpace))
		return
	}

	err = bucket.SyncTo(j.WorkSpace, "", false)
	if err != nil {
		j.addError(errors.Wrapf(err, "problem uploading %s to %s",
			j.WorkSpace, bucket))
	}
}
