package poplar

import (
	"time"

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
	PayloadFTDC           bool      `bson:"is_ftdc,omitempty" json:"is_ftdc,omitempty" yaml:"is_ftdc,omitempty"`
	PayloadBSON           bool      `bson:"is_bson,omitempty" json:"is_bson,omitempty" yaml:"is_bson,omitempty"`
	DataUncompressed      bool      `bson:"is_uncompressed" json:"is_uncompressed" yaml:"is_uncompressed"`
	DataGzipped           bool      `bson:"is_gzip,omitempty" json:"is_gzip,omitempty" yaml:"is_gzip,omitempty"`
	DataTarball           bool      `bson:"is_tarball,omitempty" json:"is_tarball,omitempty" yaml:"is_tarball,omitempty"`
	EventsRaw             bool      `bson:"events_raw,omitempty" json:"events_raw,omitempty" yaml:"events_raw,omitempty"`
	EventsHistogram       bool      `bson:"events_histogram,omitempty" json:"events_histogram,omitempty" yaml:"events_histogram,omitempty"`
	EventsIntervalSummary bool      `bson:"events_interval_summary,omitempty" json:"events_interval_summary,omitempty" yaml:"events_interval_summary,omitempty"`
	EventsCollapsed       bool      `bson:"events_collapsed,omitempty" json:"events_collapsed,omitempty" yaml:"events_collapsed,omitempty"`
}

func (a *TestArtifact) Validate() error {
	catcher := grip.NewBasicCatcher()

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
