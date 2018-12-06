package rpc

import (
	"context"

	"github.com/evergreen-ci/poplar"
	"github.com/evergreen-ci/poplar/rpc/internal"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func UploadReport(ctx context.Context, report *poplar.Report, cc *grpc.ClientConn) error {
	return errors.Wrap(uploadTests(ctx, internal.NewCedarPerformanceMetricsClient(cc), report, report.Tests),
		"problem uploading tests for report")
}

func uploadTests(ctx context.Context, client internal.CedarPerformanceMetricsClient, report *poplar.Report, tests []poplar.Test) error {
	for idx, test := range tests {
		artifacts := make([]*internal.ArtifactInfo, 0, len(test.Artifacts))
		for _, a := range test.Artifacts {
			artifacts = append(artifacts, internal.ExportArtifactInfo(&a))
		}

		resp, err := client.CreateMetricSeries(ctx, &internal.ResultData{
			Id: &internal.ResultID{
				Project:   report.Project,
				Version:   report.Version,
				TaskName:  report.TaskName,
				TaskId:    report.TaskID,
				Execution: int32(report.Execution),
				TestName:  test.Info.TestName,
				Trial:     int32(test.Info.Trial),
				Tags:      test.Info.Tags,
				Arguments: test.Info.Arguments,
				Parent:    test.Info.Parent,
			},
			Artifacts: artifacts,
		})
		if err != nil {
			return errors.Wrapf(err, "problem submitting test %d of %d", idx, len(tests))
		} else if !resp.Success {
			return errors.New("operation return failed state")
		}
		test.ID = resp.Id

		for _, st := range test.SubTests {
			st.Info.Parent = test.ID
		}

		rollups := make([]*internal.RollupValue, 0, len(test.Metrics))
		for _, r := range test.Metrics {
			rollups = append(rollups, internal.ExportRollup(&r))
		}

		resp, err = client.AttachRollups(ctx, &internal.RollupData{Id: test.ID, Rollups: rollups})
		if err != nil {
			return errors.Wrapf(err, "problem attaching rollups for '%s'", test.ID)
		}

		if err = uploadTests(ctx, client, report, test.SubTests); err != nil {
			return errors.Wrapf(err, "problem submitting subtests of '%s'", test.ID)
		}
	}

	return nil
}
