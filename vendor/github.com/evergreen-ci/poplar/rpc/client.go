package rpc

import (
	"context"
	"path/filepath"

	"github.com/evergreen-ci/poplar"
	"github.com/evergreen-ci/poplar/rpc/internal"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func UploadReport(ctx context.Context, report *poplar.Report, cc *grpc.ClientConn, dryRun bool) error {
	return errors.Wrap(uploadTests(ctx, internal.NewCedarPerformanceMetricsClient(cc), report, report.Tests, dryRun),
		"problem uploading tests for report")
}

func uploadTests(ctx context.Context, client internal.CedarPerformanceMetricsClient, report *poplar.Report, tests []poplar.Test, dryRun bool) error {
	for idx, test := range tests {
		grip.Info(message.Fields{
			"num":     idx,
			"total":   len(tests),
			"parent":  test.Info.Parent != "",
			"name":    test.Info.TestName,
			"task":    report.TaskID,
			"dry_run": dryRun,
		})

		createdAt, err := internal.ExportTimestamp(test.CreatedAt)
		if err != nil {
			return err
		}
		artifacts, err := extractArtifacts(ctx, report, test, dryRun)
		if err != nil {
			return errors.Wrap(err, "problem extracting artifacts")
		}
		resultData := &internal.ResultData{
			Id: &internal.ResultID{
				Project:   report.Project,
				Version:   report.Version,
				Variant:   report.Variant,
				TaskName:  report.TaskName,
				TaskId:    report.TaskID,
				Mainline:  report.Mainline,
				Execution: int32(report.Execution),
				TestName:  test.Info.TestName,
				Trial:     int32(test.Info.Trial),
				Tags:      test.Info.Tags,
				Arguments: test.Info.Arguments,
				Parent:    test.Info.Parent,
				CreatedAt: createdAt,
			},
			Artifacts: artifacts,
			Rollups:   extractMetrics(ctx, test),
		}

		if dryRun {
			grip.Info(message.Fields{
				"message":     "dry-run mode",
				"function":    "CreateMetricSeries",
				"result_data": resultData,
			})
		} else {
			var resp *internal.MetricsResponse
			resp, err = client.CreateMetricSeries(ctx, resultData)
			if err != nil {
				return errors.Wrapf(err, "problem submitting test %d of %d", idx, len(tests))
			} else if !resp.Success {
				return errors.New("operation return failed state")
			}

			test.ID = resp.Id
			for i := range test.SubTests {
				test.SubTests[i].Info.Parent = test.ID
			}
		}

		if err = uploadTests(ctx, client, report, test.SubTests, dryRun); err != nil {
			return errors.Wrapf(err, "problem submitting subtests of '%s'", test.ID)
		}

		completedAt, err := internal.ExportTimestamp(test.CompletedAt)
		if err != nil {
			return err
		}
		end := &internal.MetricsSeriesEnd{
			Id:          test.ID,
			IsComplete:  true,
			CompletedAt: completedAt,
		}

		if dryRun {
			grip.Info(message.Fields{
				"message":           "dry-run mode",
				"function":          "CloseMetrics",
				"metric_series_end": end,
			})
		} else {
			var resp *internal.MetricsResponse
			resp, err = client.CloseMetrics(ctx, end)
			if err != nil {
				return errors.Wrapf(err, "problem closing metrics series for '%s'", test.ID)
			} else if !resp.Success {
				return errors.New("operation return failed state")
			}
		}

	}

	return nil
}

func extractArtifacts(ctx context.Context, report *poplar.Report, test poplar.Test, dryRun bool) ([]*internal.ArtifactInfo, error) {
	artifacts := make([]*internal.ArtifactInfo, 0, len(test.Artifacts))
	for _, a := range test.Artifacts {
		if err := a.Validate(); err != nil {
			return nil, errors.Wrap(err, "problem validating artifact")
		}

		if a.LocalFile != "" {
			if a.Path == "" {
				a.Path = filepath.Join(test.ID, filepath.Base(a.LocalFile))
			}

			grip.Info(message.Fields{
				"op":     "uploading file",
				"path":   a.Path,
				"bucket": a.Bucket,
				"prefix": a.Prefix,
				"file":   a.LocalFile,
			})

			if err := a.Convert(ctx); err != nil {
				return nil, errors.Wrap(err, "problem converting artifact")
			}

			if err := a.Upload(ctx, &report.BucketConf, dryRun); err != nil {
				return nil, errors.Wrap(err, "problem uploading artifact")
			}
		}
		artifacts = append(artifacts, internal.ExportArtifactInfo(&a))
	}

	return artifacts, nil
}

func extractMetrics(ctx context.Context, test poplar.Test) []*internal.RollupValue {
	rollups := make([]*internal.RollupValue, 0, len(test.Metrics))
	for _, r := range test.Metrics {
		rollups = append(rollups, internal.ExportRollup(&r))
	}

	return rollups
}
