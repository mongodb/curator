package repobuilder

import (
	"context"
	"time"

	"github.com/evergreen-ci/pail"
	"github.com/goamz/goamz/s3"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/amboy/registry"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
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
	*job.Base `bson:"metadata" json:"metadata" yaml:"metadata"`
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
		Base: &job.Base{
			JobType: amboy.JobType{
				Name:    "build-index-pages",
				Version: 1,
			},
		},
	}

	j.SetDependency(dependency.NewAlways())

	return j
}

// Run downloads the repository, and generates index pages at all
// levels of the repo.
func (j *IndexBuildJob) Run(ctx context.Context) {
	opts := pail.S3Options{
		Region:                   j.Conf.Region,
		SharedCredentialsProfile: j.Profile,
		Name:                     j.Bucket,
		DryRun:                   j.DryRun,
		Permission:               string(s3.PublicRead),
	}
	bucket, err := pail.NewS3Bucket(opts)
	if err != nil {
		j.AddError(errors.Wrapf(err, "problem getting s3 bucket %s", j.Bucket))
		return
	}

	defer j.MarkComplete()

	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(10*time.Minute))
		defer cancel()
	}

	grip.Infof("downloading from %s to %s", bucket, j.WorkSpace)
	if err = bucket.Pull(ctx, j.WorkSpace, ""); err != nil {
		j.AddError(errors.Wrapf(err, "problem syncing from %s to %s", bucket, j.WorkSpace))
		return
	}

	if j.RepoName == "" {
		j.RepoName = j.Bucket
	}

	err = j.Conf.BuildIndexPageForDirectory(j.WorkSpace, j.RepoName)
	if err != nil {
		j.AddError(errors.Wrapf(err, "problem building index.html pages for %s", j.WorkSpace))
		return
	}

	if err = bucket.Push(ctx, j.WorkSpace, ""); err != nil {
		j.AddError(errors.Wrapf(err, "problem uploading %s to %s",
			j.WorkSpace, bucket))
	}
}
