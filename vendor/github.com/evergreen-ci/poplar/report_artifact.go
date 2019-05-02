package poplar

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"strings"

	"github.com/evergreen-ci/pail"
	"github.com/mongodb/ftdc"
	"github.com/mongodb/ftdc/bsonx"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

const defaultChunkSize = 2048

// Convert translates a the artifact into a different format,
// typically by converting JSON, BSON, or CSV to FTDC, and also
// optionally gzipping the results.
func (a *TestArtifact) Convert(ctx context.Context) error {
	if err := a.Validate(); err != nil {
		return errors.New("invalid test artifact")
	}

	if a.LocalFile == "" {
		return errors.New("cannot specify a conversion on a remote file")
	}

	if _, err := os.Stat(a.LocalFile); os.IsNotExist(err) {
		return errors.New("cannot convert non existant file")
	}

	converted := false
	switch {
	case a.ConvertBSON2FTDC:
		converted = true
		fn, err := a.bsonToFTDC(ctx, a.LocalFile)
		if err != nil {
			return errors.Wrap(err, "problem converting file")
		}
		a.LocalFile = fn
	case a.ConvertJSON2FTDC:
		converted = true
		fn, err := a.jsonToFTDC(ctx, a.LocalFile)
		if err != nil {
			return errors.Wrap(err, "problem converting file")
		}
		a.LocalFile = fn
	case a.ConvertCSV2FTDC:
		converted = true
		fn, err := a.csvToFTDC(ctx, a.LocalFile)
		if err != nil {
			return errors.Wrap(err, "problem converting file")
		}
		a.LocalFile = fn
	case a.ConvertGzip:
		converted = true
		fn, err := a.gzip(a.LocalFile)
		if err != nil {
			return errors.Wrap(err, "problem writing file")
		}
		a.LocalFile = fn
	}

	if !converted {
		grip.Warning("no conversion took place")
	}

	return nil
}

// Upload provides a way to upload an artifact using a bucket configuration.
func (a *TestArtifact) Upload(ctx context.Context, conf *BucketConfiguration) error {
	if a.LocalFile == "" {
		return errors.New("cannot upload unspecified file")
	}
	if a.Path == "" {
		return errors.New("must specify remote path")
	}
	if conf == nil {
		return errors.New("bucket configruation cannot be nil")
	}

	var err error

	if _, err = os.Stat(a.LocalFile); os.IsNotExist(err) {
		return errors.New("cannot upload file that does not exist")
	}

	if conf.bucket == nil || conf.name != a.Bucket || conf.prefix != a.Prefix {
		if a.Bucket == "" {
			return errors.New("cannot upload file, no bucket specified")
		}
		if conf.Region == "" {
			return errors.New("bucket configuration must specify a region")
		}

		opts := pail.S3Options{
			Name:       a.Bucket,
			Prefix:     a.Prefix,
			Region:     conf.Region,
			Permission: a.Permissions,
		}
		if (conf.APIKey != "" && conf.APISecret != "") || conf.APIToken != "" {
			opts.Credentials = pail.CreateAWSCredentials(conf.APIKey, conf.APISecret, conf.APIToken)
		}

		conf.bucket, err = pail.NewS3Bucket(opts)
		if err != nil {
			return errors.Wrap(err, "could not construct bucket")
		}
		conf.name = a.Bucket
		conf.prefix = a.Prefix
	}

	if err := conf.bucket.Upload(ctx, a.Path, a.LocalFile); err != nil {
		return errors.Wrap(err, "problem uploading file")
	}

	return nil
}

func (a *TestArtifact) bsonToFTDC(ctx context.Context, path string) (string, error) {
	srcFile, err := os.Open(path)
	if err != nil {
		return path, errors.Wrapf(err, "problem opening bson input file '%s'", path)
	}
	defer srcFile.Close()

	path = strings.TrimSuffix(path, ".bson") + ".ftdc"
	catcher := grip.NewCatcher()
	ftdcFile, err := os.Create(path)
	if err != nil {
		return path, errors.Wrapf(err, "problem opening ftdc output file '%s'", path)
	}
	defer func() { catcher.Add(ftdcFile.Close()) }()

	collector := ftdc.NewStreamingDynamicCollector(defaultChunkSize, ftdcFile)
	defer func() { catcher.Add(ftdc.FlushCollector(collector, ftdcFile)) }()

	for {
		if ctx.Err() != nil {
			catcher.Add(errors.New("operation aborted"))
			break
		}

		bsonDoc := bsonx.NewDocument()
		_, err = bsonDoc.ReadFrom(srcFile)
		if err != nil {
			if err == io.EOF {
				break
			}
			catcher.Add(errors.Wrap(err, "failed to read BSON"))
			break
		}

		err = collector.Add(bsonDoc)
		if err != nil {
			catcher.Add(errors.Wrap(err, "failed to write FTDC from BSON"))
			break
		}
	}

	return path, catcher.Resolve()
}

func (a *TestArtifact) csvToFTDC(ctx context.Context, path string) (string, error) {
	srcFile, err := os.Open(path)
	if err != nil {
		return path, errors.Wrapf(err, "problem opening csv input file '%s'", path)
	}
	defer srcFile.Close()

	path = strings.TrimSuffix(path, ".csv") + ".ftdc"
	catcher := grip.NewCatcher()
	ftdcFile, err := os.Create(path)
	if err != nil {
		return path, errors.Wrapf(err, "problem opening ftdc output file '%s'", path)
	}
	defer func() { catcher.Add(ftdcFile.Close()) }()

	catcher.Add(errors.Wrap(ftdc.ConvertFromCSV(ctx, defaultChunkSize, srcFile, ftdcFile),
		"problem converting csv to ftdc file"))

	return path, catcher.Resolve()
}

func (a *TestArtifact) jsonToFTDC(ctx context.Context, path string) (string, error) {
	srcFile, err := os.Open(path)
	if err != nil {
		return path, errors.Wrapf(err, "problem opening csv input file '%s'", path)
	}
	defer srcFile.Close()

	path = strings.TrimSuffix(path, ".json") + ".ftdc"
	catcher := grip.NewCatcher()
	ftdcFile, err := os.Create(path)
	if err != nil {
		return path, errors.Wrapf(err, "problem opening ftdc output file '%s'", path)
	}
	defer func() { catcher.Add(ftdcFile.Close()) }()

	opts := ftdc.CollectJSONOptions{
		OutputFilePrefix: strings.TrimSuffix(path, ".json"),
		FileName:         path,
	}
	return path, ftdc.CollectJSONStream(ctx, opts)
}

func (a *TestArtifact) gzip(path string) (string, error) {
	srcFile, err := os.Open(path)
	if err != nil {
		return path, errors.Wrapf(err, "problem opening bson input file '%s'", path)
	}
	defer srcFile.Close()

	path += ".gz"
	catcher := grip.NewCatcher()
	outFile, err := os.Create(path)
	if err != nil {
		return path, errors.Wrapf(err, "problem opening ftdc output file '%s'", path)
	}
	defer func() { catcher.Add(outFile.Close()) }()

	writer, err := gzip.NewWriterLevel(outFile, gzip.BestCompression)
	if err != nil {
		catcher.Add(err)
		return path, catcher.Resolve()
	}
	defer func() { catcher.Add(writer.Close()) }()

	_, err = io.Copy(writer, srcFile)
	catcher.Add(err)
	return path, catcher.Resolve()
}
