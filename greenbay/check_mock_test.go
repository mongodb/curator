package greenbay

import (
	"context"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
)

type mockCheck struct {
	hasRun bool
	mockCheckBase
}

func (c *mockCheck) Run(_ context.Context) {
	c.WasSuccessful = true
	c.MarkComplete()
	c.hasRun = true
}

func newMockCheckBase(id string, version int) *mockCheckBase {
	return &mockCheckBase{
		Base: job.Base{
			JobType: amboy.JobType{
				Name:    id,
				Version: version,
			},
		},
	}
}

type mockCheckBase struct { // nolint: megacheck
	WasSuccessful bool       `bson:"passed" json:"passed" yaml:"passed"`
	Message       string     `bson:"message" json:"message" yaml:"message"`
	TestSuites    []string   `bson:"suites" json:"suites" yaml:"suites"`
	Timing        TimingInfo `bson:"timing" json:"timing" yaml:"timing"`
	job.Base      `bson:"metadata" json:"metadata" yaml:"metadata"`
}

func (c *mockCheckBase) Output() CheckOutput {
	c.SetID("foo")
	out := CheckOutput{
		Name:      c.ID(),
		Check:     c.Type().Name,
		Suites:    c.Suites(),
		Completed: c.Status().Completed,
		Passed:    c.WasSuccessful,
		Message:   c.Message,
		Timing: TimingInfo{
			Start: c.Timing.Start,
			End:   c.Timing.End,
		},
	}

	if err := c.Error(); err != nil {
		out.Error = err.Error()
	}

	return out
}

func (c *mockCheckBase) Suites() []string     { return c.TestSuites }
func (c *mockCheckBase) SetSuites(s []string) { c.TestSuites = s }
func (c *mockCheckBase) Name() string         { return c.JobType.Name }
