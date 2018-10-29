package operations

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/mongodb/ftdc"
	"github.com/mongodb/grip"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func FTDC() cli.Command {
	return cli.Command{
		Name:  "ftdc",
		Usage: "tools for running FTDC parsing and generating tools",
		Flags: []cli.Flag{},
		Subcommands: []cli.Command{
			toJSON(),
			fromJSON(),
			toBSON(),
			fromBSON(),
		},
	}
}

func toJSON() cli.Command {
	return cli.Command{
		Name:  "tojson",
		Usage: "write FTDC data to a JSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "input",
				Usage: "write FTDC data from this file",
			},
			cli.StringFlag{
				Name:  "output",
				Usage: "write FTDC data in JSON format to this file, defaults to stdout",
			},
			cli.BoolFlag{
				Name:  "flattened",
				Usage: "flatten FTDC data",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ftdcPath := c.String("input")
			if ftdcPath == "" {
				return errors.New("input is not specified")
			}
			jsonPath := c.String("output")

			ftdcFile, err := os.Open(ftdcPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening file '%s'", ftdcPath)
			}
			defer ftdcFile.Close()

			var jsonFile *os.File
			if jsonPath == "" {
				jsonFile = os.Stdout
			} else {
				jsonFile, err = os.Create(jsonPath)
				if err != nil {
					return errors.Wrapf(err, "problem opening flie '%s'", jsonPath)
				}
				defer jsonFile.Close()
			}

			var iter ftdc.Iterator
			if c.Bool("flattened") {
				iter = ftdc.ReadMetrics(ctx, ftdcFile)
			} else {
				iter = ftdc.ReadStructuredMetrics(ctx, ftdcFile)
			}

			for iter.Next(ctx) {
				jsonFile.WriteString(iter.Document().ToExtJSON(false))
				jsonFile.WriteString("\n")
			}

			return iter.Err()
		},
	}
}

func fromJSON() cli.Command {
	return cli.Command{
		Name:  "fromjson",
		Usage: "write FTDC data from a JSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "input",
				Usage: "write JSON data from this file, defaults to stdin",
			},
			cli.StringFlag{
				Name:  "prefix",
				Usage: "prefix for FTDC filenames",
			},
			cli.IntFlag{
				Name:  "maxChunkSize",
				Usage: "maximum chunk size",
				Value: 1000,
			},
			cli.StringFlag{
				Name:  "flushInterval",
				Usage: "flush interval in ms",
				Value: "20",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			opts := ftdc.CollectJSONOptions{}

			jsonPath := c.String("input")
			if jsonPath == "" {
				opts.InputSource = os.Stdin
			} else {
				opts.FileName = jsonPath
			}

			ftdcPrefix := c.String("prefix")
			if ftdcPrefix == "" {
				return errors.New("prefix is not specified")
			} else {
				opts.OutputFilePrefix = ftdcPrefix
			}

			flushInterval, err := time.ParseDuration(c.String("flushInterval") + "ms")
			if err != nil {
				return errors.Wrapf(err, "failed to parse duration '%s'", c.String("flushInterval"))
			} else {
				opts.FlushInterval = flushInterval
			}

			opts.ChunkSizeBytes = c.Int("maxChunkSize")

			if err := ftdc.CollectJSONStream(ctx, opts); err != nil {
				return errors.Wrap(err, "Failed to write FTDC from JSON")
			}
			return nil
		},
	}
}

func toBSON() cli.Command {
	return cli.Command{
		Name:  "tobson",
		Usage: "write FTDC data to a BSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "input",
				Usage: "write FTDC data from this file",
			},
			cli.StringFlag{
				Name:  "output",
				Usage: "write FTDC data in BSON format to this file; defaults to stdout",
			},
			cli.BoolFlag{
				Name:  "flattened",
				Usage: "flatten FTDC data",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ftdcPath := c.String("input")
			if ftdcPath == "" {
				return errors.New("input is not specified")
			}
			bsonPath := c.String("output")

			ftdcFile, err := os.Open(ftdcPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening file '%s'", ftdcPath)
			}
			defer ftdcFile.Close()

			var bsonFile *os.File
			if bsonPath == "" {
				bsonFile = os.Stdout
			} else {
				bsonFile, err = os.Create(bsonPath)
				if err != nil {
					return errors.Wrapf(err, "problem opening flie '%s'", bsonPath)
				}
			}

			var iter ftdc.Iterator
			if c.Bool("flattened") {
				iter = ftdc.ReadMetrics(ctx, ftdcFile)
			} else {
				iter = ftdc.ReadStructuredMetrics(ctx, ftdcFile)
			}

			for iter.Next(ctx) {
				bytes, err := iter.Document().MarshalBSON()
				if err != nil {
					grip.Warning(bsonFile.Close())
					return errors.Wrap(err, "problem marshaling BSON")
				}
				bsonFile.Write(bytes)
			}

			grip.Warning(bsonFile.Close())
			return errors.Wrap(err, "problem iterating ftdc file")
		},
	}
}

func fromBSON() cli.Command {
	return cli.Command{
		Name:  "frombson",
		Usage: "write FTDC data from a BSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "input",
				Usage: "write BSON data from this file",
			},
			cli.StringFlag{
				Name:  "output",
				Usage: "write BSON data in FTDC format to this file",
			},
			cli.IntFlag{
				Name:  "maxChunkSize",
				Usage: "maximum chunk size",
				Value: 1000,
			},
		},
		Action: func(c *cli.Context) error {
			_, cancel := context.WithCancel(context.Background())
			defer cancel()

			bsonPath := c.String("input")
			if bsonPath == "" {
				return errors.New("input is not specified")
			}
			ftdcPrefix := c.String("output")
			if ftdcPrefix == "" {
				return errors.New("output is not specified")
			}
			maxChunkSize := c.Int("maxChunkSize")

			bsonFile, err := os.Open(bsonPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening flie '%s'", bsonPath)
			}
			defer bsonFile.Close()

			bsonDoc := bson.NewDocument()
			collector := ftdc.NewDynamicCollector(maxChunkSize)
			for {
				_, err := bsonDoc.ReadFrom(bsonFile)
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

			output, err := collector.Resolve()
			if err != nil {
				return errors.Wrap(err, "failed to write FTDC from BSON")
			}

			if err = ioutil.WriteFile(ftdcPrefix, output, 0600); err != nil {
				return errors.Wrap(err, "failed to write FTDC from BSON")
			}
			return nil
		},
	}
}
