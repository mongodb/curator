package operations

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mongodb/ftdc"
	"github.com/mongodb/grip/recovery"
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
			ftdctojson(),
			jsontoftdc(),
			ftdctobson(),
			bsontoftdc(),
			sysInfoCollector(),
		},
	}
}

func ftdctojson() cli.Command {
	return cli.Command{
		Name:  "ftdctojson",
		Usage: "write FTDC data to a JSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "ftdcPath",
				Usage: "write FTDC data from this file",
			},
			cli.StringFlag{
				Name:  "jsonPath",
				Usage: "write FTDC data in JSON format to this file; if no file is specified, defaults to standard out",
			},
			cli.BoolFlag{
				Name:  "flattened",
				Usage: "flatten FTDC data",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ftdcPath := c.String("ftdcPath")
			if ftdcPath == "" {
				return errors.New("ftdcPath is not specified")
			}
			jsonPath := c.String("jsonPath")

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

// add maxChunkSize flag
func jsontoftdc() cli.Command {
	return cli.Command{
		Name:  "jsontoftdc",
		Usage: "write FTDC data from a JSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "jsonPath",
				Usage: "JSON filepath",
			},
			cli.StringFlag{
				Name:  "ftdcPrefix",
				Usage: "prefix for FTDC filenames",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			jsonPath := c.String("jsonPath")
			if jsonPath == "" {
				return errors.New("jsonPath is not specified")
			}
			ftdcPrefix := c.String("ftdcPrefix")
			if ftdcPrefix == "" {
				return errors.New("ftdcPrefix is not specified")
			}

			opts := ftdc.CollectJSONOptions{FileName: jsonPath, OutputFilePrefix: ftdcPrefix}
			if err := ftdc.CollectJSONStream(ctx, opts); err != nil {
				return errors.Wrap(err, "Failed to write FTDC from JSON")
			}
			return nil
		},
	}
}

func ftdctobson() cli.Command {
	return cli.Command{
		Name:  "ftdctobson",
		Usage: "write FTDC data to a BSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "ftdcPath",
				Usage: "write FTDC data from this file",
			},
			cli.StringFlag{
				Name:  "bsonPath",
				Usage: "write FTDC data in BSON format to this file; if no file is specified, defaults to standard out",
			},
			cli.BoolFlag{
				Name:  "flattened",
				Usage: "flatten FTDC data",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ftdcPath := c.String("ftdcPath")
			if ftdcPath == "" {
				return errors.New("ftdcPath is not specified")
			}
			bsonPath := c.String("bsonPath")

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
				defer bsonFile.Close()
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
					return errors.Wrap(err, "problem marshaling BSON")
				}
				bsonFile.Write(bytes)
			}
			return iter.Err()
		},
	}
}

func bsontoftdc() cli.Command {
	return cli.Command{
		Name:  "bsontoftdc",
		Usage: "write FTDC data from a BSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "bsonPath",
				Usage: "BSON filepath",
			},
			cli.StringFlag{
				Name:  "ftdcPrefix",
				Usage: "prefix for FTDC filenames",
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

			bsonPath := c.String("bsonPath")
			if bsonPath == "" {
				return errors.New("bsonPath is not specified")
			}
			ftdcPrefix := c.String("ftdcPrefix")
			if ftdcPrefix == "" {
				return errors.New("ftdcPrefix is not specified")
			}
			maxChunkSize := c.Int("maxChunkSize")

			bsonFile, err := os.Open(bsonPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening flie '%s'", bsonPath)
			}
			defer bsonFile.Close()

			var bsonDoc *bson.Document
			collector := ftdc.NewDynamicCollector(maxChunkSize)
			for {
				_, err = bsonDoc.ReadFrom(bsonFile)
				if err != nil {
					fmt.Println(err)
					break
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

			ftdcFile, err := os.Create(ftdcPrefix)
			if err != nil {
				return errors.Wrapf(err, "problem opening file '%s'", ftdcPrefix)
			}
			defer ftdcFile.Close()

			_, err = ftdcFile.Write(output)
			if err != nil {
				return errors.Wrap(err, "failed to write FTDC from BSON")
			}
			return nil
		},
	}
}

func sysInfoCollector() cli.Command {
	return cli.Command{
		Name:  "sysinfo-collector",
		Usage: "collect system info metrics",
		Flags: []cli.Flag{
			cli.DurationFlag{
				Name:  "interval",
				Usage: "interval to collect system info metrics",
				Value: time.Second,
			},
			cli.DurationFlag{
				Name:  "flush",
				Usage: "interval to flush data to file",
				Value: 4 * time.Hour,
			},
			cli.StringFlag{
				Name:  "prefix",
				Usage: "prefix for FTDC filenames",
				Value: fmt.Sprintf("sysinfo.%s", time.Now().Format("2006-01-02.15-04-05")),
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			interval := c.Duration("interval")
			flush := c.Duration("flush")
			prefix := c.String("prefix")

			opts := ftdc.CollectSysInfoOptions{
				ChunkSizeBytes:     math.MaxInt32,
				OutputFilePrefix:   prefix,
				FlushInterval:      flush,
				CollectionInterval: interval,
			}

			go signalListenner(ctx, cancel)
			return ftdc.CollectSysInfo(ctx, opts)
		},
	}
}

func signalListenner(ctx context.Context, trigger context.CancelFunc) {
	defer recovery.LogStackTraceAndContinue("graceful shutdown")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	<-sigChan
	trigger()
}
