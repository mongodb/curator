package operations

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/mongodb/ftdc"
	"github.com/mongodb/ftdc/bsonx"
	"github.com/mongodb/grip"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// flag names
var (
	input     = "input"
	output    = "output"
	prefix    = "prefix"
	flattened = "flattened"
	maxCount  = "maxCount"
	flush     = "flush"
)

func FTDC() cli.Command {
	return cli.Command{
		Name:  "ftdc",
		Usage: "tools for manipulating FTDC data",
		Subcommands: []cli.Command{
			cli.Command{
				Name:  "export",
				Usage: "write FTDC data to other encoding formats",
				Subcommands: []cli.Command{
					toJSON(),
					toBSON(),
					toCSV(),
				},
			},
			cli.Command{
				Name:  "import",
				Usage: "compress data in FTDC format",
				Subcommands: []cli.Command{
					fromJSON(),
					fromBSON(),
					fromCSV(),
				},
			},
		},
	}
}

func toJSON() cli.Command {
	return cli.Command{
		Name:  "json",
		Usage: "write data from an FTDC file data to JSON",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  input,
				Usage: "source FTDC data file",
			},
			cli.StringFlag{
				Name:  output,
				Usage: "write FTDC data in JSON format to this file (default: stdout)",
			},
			cli.BoolFlag{
				Name:  flattened,
				Usage: "flatten FTDC data",
			},
		},
		Before: requireFileExists(input, false),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ftdcPath := c.String(input)
			jsonPath := c.String(output)

			ftdcFile, err := os.Open(ftdcPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening file '%s'", ftdcPath)
			}
			defer ftdcFile.Close()

			var jsonFile *os.File
			if jsonPath == "" {
				jsonFile = os.Stdout
			} else {
				if _, err = os.Stat(jsonPath); !os.IsNotExist(err) {
					return errors.Errorf("cannot export bson to %s, file already exists", jsonPath)
				}

				jsonFile, err = os.Create(jsonPath)
				if err != nil {
					return errors.Wrapf(err, "problem opening flie '%s'", jsonPath)
				}
				defer jsonFile.Close()
			}

			var iter ftdc.Iterator
			if c.Bool(flattened) {
				iter = ftdc.ReadMetrics(ctx, ftdcFile)
			} else {
				iter = ftdc.ReadStructuredMetrics(ctx, ftdcFile)
			}

			for iter.Next() {
				doc, err := bson.MarshalExtJSON(iter.Document(), false, false)
				if err != nil {
					return errors.Wrap(err, "problem reading document to json")
				}

				jsonFile.WriteString(string(doc))
				jsonFile.WriteString("\n")
			}

			return iter.Err()
		},
	}
}

func fromJSON() cli.Command {
	return cli.Command{
		Name:  "json",
		Usage: "write data from new line seperated JSON to FTDC files",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  input,
				Usage: "write JSON data from this file (default: stdin)",
			},
			cli.StringFlag{
				Name:  prefix,
				Usage: "prefix for FTDC filenames",
			},
			cli.IntFlag{
				Name:  maxCount,
				Usage: "maximum number of samples per chunk",
				Value: 1000,
			},
			cli.DurationFlag{
				Name:  flush,
				Usage: "flush interval",
				Value: 20 * time.Millisecond,
			},
		},
		Before: mergeBeforeFuncs(
			requireFileExists(input, true),
			requireStringFlag(prefix),
		),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			opts := ftdc.CollectJSONOptions{}

			jsonPath := c.String(input)
			if jsonPath == "" {
				opts.InputSource = os.Stdin
			} else {
				opts.FileName = jsonPath
			}
			opts.OutputFilePrefix = c.String(prefix)
			opts.FlushInterval = c.Duration(flush)
			opts.SampleCount = c.Int(maxCount)

			if err := ftdc.CollectJSONStream(ctx, opts); err != nil {
				return errors.Wrap(err, "Failed to write FTDC from JSON")
			}
			return nil
		},
	}
}

func toBSON() cli.Command {
	return cli.Command{
		Name:  "bson",
		Usage: "write data from an FTDC file to a BSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  input,
				Usage: "read source FTDC data from this file",
			},
			cli.StringFlag{
				Name:  output,
				Usage: "write BSON data to this file (default: stdout)",
			},
			cli.BoolFlag{
				Name:  flattened,
				Usage: "flatten document structure data",
			},
		},
		Before: mergeBeforeFuncs(
			requireFileExists(input, false),
		),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ftdcPath := c.String(input)
			bsonPath := c.String(output)

			ftdcFile, err := os.Open(ftdcPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening file '%s'", ftdcPath)
			}
			defer ftdcFile.Close()

			var bsonFile *os.File
			if bsonPath == "" {
				bsonFile = os.Stdout
			} else {
				if _, err = os.Stat(bsonPath); !os.IsNotExist(err) {
					return errors.Errorf("cannot export ftdc to %s, file already exists", bsonPath)
				}

				bsonFile, err = os.Create(bsonPath)
				if err != nil {
					return errors.Wrapf(err, "problem opening flie '%s'", bsonPath)
				}
				defer func() { grip.Warning(bsonFile.Close()) }()
			}

			var iter ftdc.Iterator
			if c.Bool(flattened) {
				iter = ftdc.ReadMetrics(ctx, ftdcFile)
			} else {
				iter = ftdc.ReadStructuredMetrics(ctx, ftdcFile)
			}

			for iter.Next() {
				bytes, err := iter.Document().MarshalBSON()
				if err != nil {
					return errors.Wrap(err, "problem marshaling BSON")
				}
				_, err = bsonFile.Write(bytes)
				if err != nil {
					return errors.Wrap(err, "problem writing data to file")
				}
			}

			return errors.Wrap(err, "problem iterating ftdc file")
		},
	}
}

func fromBSON() cli.Command {
	return cli.Command{
		Name:  "bson",
		Usage: "write from a BSON file to an FTDC file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  input,
				Usage: "write source BSON data from this file",
			},
			cli.StringFlag{
				Name:  output,
				Usage: "write BSON data in FTDC format to this file, which must not exist yet",
			},
			cli.IntFlag{
				Name:  maxCount,
				Usage: "maximum number of samples per chunk",
				Value: 1000,
			},
		},
		Before: mergeBeforeFuncs(
			requireFileExists(input, false),
			requireStringFlag(output),
		),
		Action: func(c *cli.Context) error {
			bsonPath := c.String(input)
			ftdcPath := c.String(output)
			maxCount := c.Int(maxCount)

			// access the source file
			//
			bsonFile, err := os.Open(bsonPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening file '%s'", bsonPath)
			}
			defer bsonFile.Close()

			// prepare the output file
			//
			if _, err := os.Stat(ftdcPath); !os.IsNotExist(err) {
				return errors.Errorf("cannot export ftdc to %s, file already exists", ftdcPath)
			}
			ftdcFile, err := os.Create(ftdcPath)
			if err != nil {
				return errors.Wrapf(err, "problem creating file '%s'", ftdcPath)
			}
			defer func() {
				grip.EmergencyFatal(errors.Wrapf(ftdcFile.Close(), "problem closing '%s' file", ftdcPath))
			}()

			// collect the data
			//
			collector := ftdc.NewStreamingDynamicCollector(maxCount, ftdcFile)
			defer func() { grip.EmergencyFatal(ftdc.FlushCollector(collector, ftdcFile)) }()
			for {
				bsonDoc := bsonx.NewDocument()
				_, err = bsonDoc.ReadFrom(bsonFile)
				if err != nil {
					if err == io.EOF {
						break
					}
					return errors.Wrap(err, "failed to write FTDC from BSON")
				}
				err = collector.Add(bsonDoc)
				if err != nil {
					return errors.Wrap(err, "failed to write FTDC from BSON")
				}
			}

			return nil
		},
	}
}

func toCSV() cli.Command {
	return cli.Command{
		Name:  "csv",
		Usage: "write data from an FTDC file to CSV",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  input,
				Usage: "source FTDC data file",
			},
			cli.StringFlag{
				Name:  output,
				Usage: "write FTDC data in CSV format to this file (default: stdout)",
			},
		},
		Before: requireFileExists(input, false),
		Action: func(c *cli.Context) error {
			inputPath := c.String(input)
			outputPath := c.String(output)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Perpare the input
			//
			inputFile, err := os.Open(inputPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening file '%s'", inputPath)
			}
			defer inputFile.Close()

			// open the data source
			//
			var outputFile *os.File
			if outputPath == "" {
				outputFile = os.Stdout
			} else {
				if _, err = os.Stat(outputPath); !os.IsNotExist(err) {
					return errors.Errorf("cannot write ftdc to '%s', file already exists", outputPath)
				}

				outputFile, err = os.Create(outputPath)
				if err != nil {
					return errors.Wrapf(err, "problem opening file '%s'", outputPath)
				}
				defer func() { grip.EmergencyFatal(outputFile.Close()) }()
			}

			// actually convert data
			//
			if err := ftdc.WriteCSV(ctx, ftdc.ReadChunks(ctx, inputFile), outputFile); err != nil {
				return errors.Wrap(err, "problem parsing csv")
			}

			return nil
		},
	}
}

func fromCSV() cli.Command {
	return cli.Command{
		Name:  "csv",
		Usage: "write data from a CSV source to an FTDC file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  input,
				Usage: "write source BSON data from this file (default: stdin)",
			},
			cli.StringFlag{
				Name:  output,
				Usage: "write CSV data in FTDC format to this file",
			},
			cli.IntFlag{
				Name:  maxCount,
				Usage: "maximum number of samples per chunk",
				Value: 1000,
			},
		},
		Before: mergeBeforeFuncs(
			requireFileExists(input, true),
			requireStringFlag(output),
		),
		Action: func(c *cli.Context) error {
			srcPath := c.String(input)
			outputPath := c.String(output)
			chunkSize := c.Int(maxCount)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Pepare the data source
			//
			var srcFile *os.File
			if srcPath == "" {
				srcFile = os.Stdin
			} else {
				var err error
				srcFile, err = os.Open(srcPath)
				if err != nil {
					return errors.Wrapf(err, "problem opening file %s", srcPath)
				}
				defer srcFile.Close()
			}

			// Create the output file
			//
			if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
				return errors.Errorf("cannot export ftdc to %s, file already exists", outputPath)
			}
			outputFile, err := os.Create(outputPath)
			if err != nil {
				return errors.Wrapf(err, "problem creating file '%s'", outputPath)
			}
			defer func() {
				grip.EmergencyFatal(errors.Wrapf(outputFile.Close(), "problem closing '%s' file", outputPath))
			}()

			// Use the library conversion function.
			//
			return errors.Wrap(ftdc.ConvertFromCSV(ctx, chunkSize, srcFile, outputFile), "problem writing data to csv")
		},
	}
}
