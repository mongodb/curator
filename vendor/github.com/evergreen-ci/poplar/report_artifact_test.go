package poplar

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/evergreen-ci/pail"
	"github.com/mongodb/ftdc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	for _, test := range []struct {
		name              string
		artifact          *TestArtifact
		expectedLocalFile string
		conversionCheck   func(io.Reader) bool
		hasErr            bool
	}{
		{
			name: "NoConversion",
			artifact: &TestArtifact{
				LocalFile: "testdata/bson_example.bson",
			},
			expectedLocalFile: "testdata/bson_example.bson",
		},
		{
			name: "IncompatibleConversions",
			artifact: &TestArtifact{
				LocalFile:        "testdata/bson_example.bson",
				ConvertBSON2FTDC: true,
				ConvertJSON2FTDC: true,
				ConvertCSV2FTDC:  true,
				ConvertGzip:      true,
			},
			expectedLocalFile: "testdata/bson_example.bson",
			hasErr:            true,
		},
		{
			name:     "NoLocalFile",
			artifact: &TestArtifact{ConvertBSON2FTDC: true},
			hasErr:   true,
		},
		{
			name: "NonExistentLocalFile",
			artifact: &TestArtifact{
				LocalFile:        "DNE",
				ConvertBSON2FTDC: true,
			},
			expectedLocalFile: "DNE",
			hasErr:            true,
		},
		{
			name: "ConvertBSON2FTDC",
			artifact: &TestArtifact{
				LocalFile:        "testdata/bson_example.bson",
				ConvertBSON2FTDC: true,
			},
			expectedLocalFile: "testdata/bson_example.ftdc",
			conversionCheck:   isFTDC,
		},
		{
			name: "ConvertJSON2FTDC",
			artifact: &TestArtifact{
				LocalFile:        "testdata/json_example.json",
				ConvertJSON2FTDC: true,
			},
			expectedLocalFile: "testdata/json_example.ftdc",
			conversionCheck:   isFTDC,
		},
		{
			name: "ConvertCSV2FTDC",
			artifact: &TestArtifact{
				LocalFile:       "testdata/csv_example.csv",
				ConvertCSV2FTDC: true,
			},
			expectedLocalFile: "testdata/csv_example.ftdc",
			conversionCheck:   isFTDC,
		},
		{
			name: "ConvertGzip",
			artifact: &TestArtifact{
				LocalFile:   "testdata/json_example.json",
				ConvertGzip: true,
			},
			expectedLocalFile: "testdata/json_example.json.gz",
			conversionCheck:   isGzipped,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.TODO()
			if test.hasErr {
				require.Error(t, test.artifact.Convert(ctx))
			} else {
				require.NoError(t, test.artifact.Convert(ctx))
			}
			assert.Equal(t, test.expectedLocalFile, test.artifact.LocalFile)
			if test.conversionCheck != nil {
				defer func() {
					assert.NoError(t, os.Remove(test.artifact.LocalFile))
				}()

				f, err := os.Open(test.artifact.LocalFile)
				require.NoError(t, err)
				assert.True(t, test.conversionCheck(f))
			}
		})
	}
}

func TestUpload(t *testing.T) {
	ctx := context.TODO()

	s3Name := "build-test-curator"
	s3Prefix := "poplar-test"
	s3Region := "us-east-1"
	for _, test := range []struct {
		name        string
		artifact    *TestArtifact
		bucketConf  BucketConfiguration
		dryRunNoErr bool
		hasErr      bool
	}{
		{
			name: "NoLocalFile",
			artifact: &TestArtifact{
				Bucket: "bucket",
				Path:   "bson_example.bson",
			},
			bucketConf: BucketConfiguration{
				Region: s3Region,
			},
			hasErr: true,
		},
		{
			name: "NoRemotePath",
			artifact: &TestArtifact{
				Bucket:    "bucket",
				LocalFile: "testdata/bson_example.bson",
			},
			bucketConf: BucketConfiguration{
				Region: s3Region,
			},
			hasErr: true,
		},
		{
			name: "NilBucketConfiguration",
			artifact: &TestArtifact{
				Bucket:    "bucket",
				Path:      "bson_example.bson",
				LocalFile: "testdata/bson_example.bson",
			},
			hasErr: true,
		},
		{
			name: "NonExistentLocalFile",
			artifact: &TestArtifact{
				Bucket:    "bucket",
				Path:      "bson_example.bson",
				LocalFile: "DNE",
			},
			bucketConf: BucketConfiguration{
				Region: s3Region,
			},
			hasErr: true,
		},
		{
			name: "NoBucketSpecified",
			artifact: &TestArtifact{
				Path:      "bson_example.bson",
				LocalFile: "testdata/bson_example.bson",
			},
			bucketConf: BucketConfiguration{
				Region: s3Region,
			},
			hasErr: true,
		},
		{
			name: "NoRegionSpecified",
			artifact: &TestArtifact{
				Bucket:    s3Name,
				Prefix:    s3Prefix,
				Path:      "bson_example.bson",
				LocalFile: "testdata/bson_example.bson",
			},
			bucketConf: BucketConfiguration{},
			hasErr:     true,
		},
		{
			name: "BadCredentialsKeyAndSecret",
			artifact: &TestArtifact{
				Bucket:    s3Name,
				Prefix:    s3Prefix,
				Path:      "bson_example.bson",
				LocalFile: "testdata/bson_example.bson",
			},
			bucketConf: BucketConfiguration{
				APIKey:    "asdf",
				APISecret: "asdf",
				Region:    s3Region,
			},
			dryRunNoErr: true,
			hasErr:      true,
		},
		{
			name: "BadCredentialsToken",
			artifact: &TestArtifact{
				Bucket:    s3Name,
				Prefix:    s3Prefix,
				Path:      "bson_example.bson",
				LocalFile: "testdata/bson_example.bson",
			},
			bucketConf: BucketConfiguration{
				APIToken: "asdf",
				Region:   s3Region,
			},
			hasErr: true,
		},
		{
			name: "SuccessfulUpload",
			artifact: &TestArtifact{
				Bucket:    s3Name,
				Prefix:    s3Prefix,
				Path:      "bson_example2.bson",
				LocalFile: "testdata/bson_example.bson",
			},
			bucketConf: BucketConfiguration{
				Region: s3Region,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.hasErr {
				if !test.dryRunNoErr {
					require.Error(t, test.artifact.Upload(ctx, test.bucketConf, true))
				}
				require.Error(t, test.artifact.Upload(ctx, test.bucketConf, false))
			} else {
				for _, dryRun := range []bool{true, false} {
					opts := pail.S3Options{
						Name:   test.artifact.Bucket,
						Prefix: test.artifact.Prefix,
						Region: test.bucketConf.Region,
					}
					if (test.bucketConf.APIKey != "" && test.bucketConf.APISecret != "") || test.bucketConf.APIToken != "" {
						opts.Credentials = pail.CreateAWSCredentials(
							test.bucketConf.APIKey,
							test.bucketConf.APISecret,
							test.bucketConf.APIToken,
						)
					}
					bucket, err := pail.NewS3Bucket(opts)
					require.NoError(t, err)

					require.NoError(t, test.artifact.Upload(ctx, test.bucketConf, dryRun))
					defer func() {
						assert.NoError(t, bucket.Remove(ctx, test.artifact.Path))
					}()

					r, err := bucket.Get(ctx, test.artifact.Path)
					if dryRun {
						require.Error(t, err)
					} else {

						require.NoError(t, err)
						remoteData, err := ioutil.ReadAll(r)
						require.NoError(t, err)
						f, err := os.Open(test.artifact.LocalFile)
						require.NoError(t, err)
						localData, err := ioutil.ReadAll(f)
						require.NoError(t, err)
						assert.Equal(t, localData, remoteData)
					}
				}
			}
		})
	}
}

func isFTDC(r io.Reader) bool {
	iter := ftdc.ReadMetrics(context.TODO(), r)
	for iter.Next() {
	}

	return iter.Err() == nil
}

func isGzipped(r io.Reader) bool {
	buff := make([]byte, 512)
	_, _ = r.Read(buff)
	return http.DetectContentType(buff) == "application/x-gzip"
}
