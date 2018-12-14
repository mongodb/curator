package poplar

import (
	"time"

	"github.com/evergreen-ci/pail"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

type Report struct {
	// These settings are at the top level to provide a DRY
	// location for the data, in the DB they're part of the
	// test-info, but we're going to assume that these tasks run
	// in Evergreen conventionally.
	Project   string `bson:"project" json:"project" yaml:"project"`
	Version   string `bson:"version" json:"version" yaml:"version"`
	Variant   string `bson:"variant" json:"variant" yaml:"variant"`
	TaskName  string `bson:"task_name" json:"task_name" yaml:"task_name"`
	TaskID    string `bson:"task_id" json:"task_id" yaml:"task_id"`
	Execution int    `bson:"execution_number" json:"execution_number" yaml:"execution_number"`

	BucketConf BucketConfiguration `bson:"bucket" json:"bucket" yaml:"bucket"`

	// Tests holds all of the test data.
	Tests []Test `bson:"tests" json:"tests" yaml:"tests"`
}

type Test struct {
	ID          string         `bson:"_id" json:"id" yaml:"id"`
	Info        TestInfo       `bson:"info" json:"info" yaml:"info"`
	CreatedAt   time.Time      `bson:"created_at" json:"created_at" yaml:"created_at"`
	CompletedAt time.Time      `bson:"completed_at" json:"completed_at" yaml:"completed_at"`
	Artifacts   []TestArtifact `bson:"artifacts" json:"artifacts" yaml:"artifacts"`
	Metrics     []TestMetrics  `bson:"metrics" json:"metrics" yaml:"metrics"`
	SubTests    []Test         `bson:"sub_tests" json:"sub_tests" yaml:"sub_tests"`
}

type TestInfo struct {
	TestName  string           `bson:"test_name" json:"test_name" yaml:"test_name"`
	Trial     int              `bson:"trial" json:"trial" yaml:"trial"`
	Parent    string           `bson:"parent" json:"parent" yaml:"parent"`
	Tags      []string         `bson:"tags" json:"tags" yaml:"tags"`
	Arguments map[string]int32 `bson:"args" json:"args" yaml:"args"`
}

type TestArtifact struct {
	Bucket                string    `bson:"bucket" json:"bucket" yaml:"bucket"`
	Path                  string    `bson:"path" json:"path" yaml:"path"`
	Tags                  []string  `bson:"tags" json:"tags" yaml:"tags"`
	CreatedAt             time.Time `bson:"created_at" json:"created_at" yaml:"created_at"`
	LocalFile             string    `bson:"local_path,omitempty" json:"local_path,omitempty" yaml:"local_path,omitempty"`
	PayloadFTDC           bool      `bson:"is_ftdc,omitempty" json:"is_ftdc,omitempty" yaml:"is_ftdc,omitempty"`
	PayloadBSON           bool      `bson:"is_bson,omitempty" json:"is_bson,omitempty" yaml:"is_bson,omitempty"`
	DataUncompressed      bool      `bson:"is_uncompressed" json:"is_uncompressed" yaml:"is_uncompressed"`
	DataGzipped           bool      `bson:"is_gzip,omitempty" json:"is_gzip,omitempty" yaml:"is_gzip,omitempty"`
	DataTarball           bool      `bson:"is_tarball,omitempty" json:"is_tarball,omitempty" yaml:"is_tarball,omitempty"`
	EventsRaw             bool      `bson:"events_raw,omitempty" json:"events_raw,omitempty" yaml:"events_raw,omitempty"`
	EventsHistogram       bool      `bson:"events_histogram,omitempty" json:"events_histogram,omitempty" yaml:"events_histogram,omitempty"`
	EventsIntervalSummary bool      `bson:"events_interval_summary,omitempty" json:"events_interval_summary,omitempty" yaml:"events_interval_summary,omitempty"`
	EventsCollapsed       bool      `bson:"events_collapsed,omitempty" json:"events_collapsed,omitempty" yaml:"events_collapsed,omitempty"`
	ConvertGzip           bool      `bson:"convert_gzip,omitempty" json:"convert_gzip,omitempty" yaml:"convert_gzip,omitempty"`
	ConvertBSON2FTDC      bool      `bson:"convert_bson_to_ftdc,omitempty" json:"convert_bson_to_ftdc,omitempty" yaml:"convert_bson_to_ftdc,omitempty"`
	ConvertCSV2FTDC       bool      `bson:"convert_csv_to_ftdc" json:"convert_csv_to_ftdc" yaml:"convert_csv_to_ftdc"`
}

func (a *TestArtifact) Validate() error {
	catcher := grip.NewBasicCatcher()

	if a.ConvertGzip {
		a.DataGzipped = true
	}

	if a.ConvertCSV2FTDC {
		a.PayloadFTDC = true

	}
	if a.ConvertBSON2FTDC {
		a.PayloadBSON = false
		a.PayloadFTDC = true
	}

	if isMoreThanOneTrue([]bool{a.ConvertBSON2FTDC, a.ConvertCSV2FTDC}) {
		catcher.Add(errors.New("cannot specify contradictory conversion requests"))
	}

	if isMoreThanOneTrue([]bool{a.PayloadBSON, a.PayloadFTDC}) {
		catcher.Add(errors.New("must specify exactly one payload type"))
	}

	if isMoreThanOneTrue([]bool{a.DataGzipped, a.DataTarball, a.DataUncompressed}) {
		catcher.Add(errors.New("must specify exactly one file format type"))
	}

	if isMoreThanOneTrue([]bool{a.EventsCollapsed, a.EventsHistogram, a.EventsIntervalSummary, a.EventsRaw}) {
		catcher.Add(errors.New("must specify exactly one event format type"))
	}

	return catcher.Resolve()
}

type TestMetrics struct {
	Name    string      `bson:"name" json:"name" yaml:"name"`
	Version int         `bson:"version,omitempty" json:"version,omitempty" yaml:"version,omitempty"`
	Type    string      `bson:"type" json:"type" yaml:"type"`
	Value   interface{} `bson:"value" json:"value" yaml:"value"`
}

type BucketConfiguration struct {
	APIKey    string `bson:"api_key" json:"api_key" yaml:"api_key"`
	APISecret string `bson:"api_secret" json:"api_secret" yaml:"api_secret"`
	APIToken  string `bson:"api_token" json:"api_token" yaml:"api_token"`
	Region    string `bson:"region" json:"region" yaml:"region"`
	Name      string `bson:"name" json:"name" yaml:"name"`
	Prefix    string `bson:"prefix" json:"prefix" yaml:"prefix"`

	bucket pail.Bucket
}
