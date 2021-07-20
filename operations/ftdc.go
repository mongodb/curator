package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/evergreen-ci/birch"
	"github.com/mongodb/ftdc"
	"github.com/mongodb/ftdc/metrics"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// FTDC command line function.
func FTDC() cli.Command {
	return cli.Command{
		Name:  "ftdc",
		Usage: "tools for manipulating FTDC data",
		Subcommands: []cli.Command{
			{
				Name:  "export",
				Usage: "write FTDC data to other encoding formats",
				Subcommands: []cli.Command{
					toJSON(),
					toBSON(),
					toCSV(),
					toMDB(),
					toT2(),
				},
			},
			{
				Name:  "import",
				Usage: "compress data in FTDC format",
				Subcommands: []cli.Command{
					fromJSON(),
					fromBSON(),
					fromCSV(),
					fromMDB(),
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
			defer func() { grip.Warning(ftdcFile.Close()) }()

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
				defer func() { grip.Warning(jsonFile.Close()) }()
			}

			var iter ftdc.Iterator
			if c.Bool(flattened) {
				iter = ftdc.ReadMetrics(ctx, ftdcFile)
			} else {
				iter = ftdc.ReadStructuredMetrics(ctx, ftdcFile)
			}

			for iter.Next() {
				dmap := iter.Document().ExportMap()
				jsondoc, err := json.Marshal(dmap)
				if err != nil {
					return errors.Wrap(err, "problem reading document to json")
				}

				_, err = jsonFile.WriteString(string(jsondoc) + "\n")
				if err != nil {
					return errors.Wrap(err, "problem writing json to document")
				}
			}

			return iter.Err()
		},
	}
}

func fromJSON() cli.Command {
	return cli.Command{
		Name:  "json",
		Usage: "write data from new line separated JSON to FTDC files",
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

			opts := metrics.CollectJSONOptions{}

			jsonPath := c.String(input)
			if jsonPath == "" {
				opts.InputSource = os.Stdin
			} else {
				opts.FileName = jsonPath
			}
			opts.OutputFilePrefix = c.String(prefix)
			opts.FlushInterval = c.Duration(flush)
			opts.SampleCount = c.Int(maxCount)

			if _, err := metrics.CollectJSONStream(ctx, opts); err != nil {
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
			defer func() { grip.Warning(ftdcFile.Close()) }()

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
				var bytes []byte
				bytes, err = iter.Document().MarshalBSON()
				if err != nil {
					return errors.Wrap(err, "problem marshaling BSON")
				}
				_, err = bsonFile.Write(bytes)
				if err != nil {
					return errors.Wrap(err, "problem writing data to file")
				}
			}

			return errors.Wrap(iter.Err(), "problem iterating ftdc file")
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
			defer func() { grip.Warning(bsonFile.Close()) }()

			// prepare the output file
			//
			if _, err = os.Stat(ftdcPath); !os.IsNotExist(err) {
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
				bsonDoc := birch.NewDocument()
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
			defer func() { grip.Warning(inputFile.Close()) }()

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
				defer func() { grip.Warning(srcFile.Close()) }()
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

func toMDB() cli.Command {
	return cli.Command{
		Name:  "mongodb",
		Usage: "load data from an FTDC file into a MongoDB database",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  input,
				Usage: "load source FTDC data from this file",
			},
			cli.StringFlag{
				Name:  "url",
				Usage: "specify the mongodb url",
				Value: "mongodb://127.0.0.1:27017",
			},
			cli.StringFlag{
				Name:  "collection, coll, c",
				Usage: "specify the name of the collection to load data too",
				Value: fmt.Sprintf("ftdc.%d", time.Now().Unix()),
			},
			cli.StringFlag{
				Name:  "database, db, d",
				Value: "curator",
			},
			cli.IntFlag{
				Name:  "batchSize",
				Value: 1000,
			},
			cli.BoolFlag{
				Name:  "drop",
				Usage: "specify 'drop' if you want to drop the collection before loading into it",
			},
			cli.BoolFlag{
				Name:  flattened,
				Usage: "flatten document structure data",
			},
			cli.BoolFlag{
				Name:  "continue",
				Usage: "continue on error during insertion",
			},
		},
		Action: func(c *cli.Context) error {
			srcPath := c.String(input)
			dbName := c.String("database")
			mdburl := c.String("url")
			collName := c.String("collection")
			batchSize := c.Int("batchSize")
			doDrop := c.Bool("drop")
			shouldContinue := c.Bool("continue")
			shouldFlatten := c.Bool(flattened)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			srcFile, err := os.Open(srcPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening file %s", srcPath)
			}
			defer func() { grip.Error(srcFile.Close()) }()

			client, err := mongo.NewClient(options.Client().ApplyURI(mdburl))
			if err != nil {
				return errors.Wrap(err, "problem creating mongodb client")
			}

			connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := client.Connect(connCtx); err != nil {
				return errors.Wrap(err, "problem connecting to mongodb")
			}

			var iter ftdc.Iterator
			if shouldFlatten {
				iter = ftdc.ReadMetrics(ctx, srcFile)
			} else {
				iter = ftdc.ReadStructuredMetrics(ctx, srcFile)
			}

			coll := client.Database(dbName).Collection(collName)
			if doDrop {
				if err := coll.Drop(ctx); err != nil {
					return errors.Wrap(err, "problem dropping collection")
				}
				grip.Infof("dropped '%s.%s'", dbName, collName)

			}

			batch := []interface{}{}
			catcher := grip.NewBasicCatcher()
			count := 0
			batches := 0
			for iter.Next() {
				batch = append(batch, iter.Document())

				if len(batch) < batchSize {
					continue
				}
				batches++

				res, err := coll.InsertMany(ctx, batch)
				batch = []interface{}{}
				catcher.Add(err)
				if res != nil {
					count += len(res.InsertedIDs)
				}
				if !shouldContinue && err != nil {
					break
				}
			}
			catcher.Add(errors.Wrapf(iter.Err(), "problem iterating ftdc data from file '%s'", srcPath))

			if len(batch) > 0 {
				batches++
				res, err := coll.InsertMany(ctx, batch)
				catcher.Add(err)
				if res != nil {
					count += len(res.InsertedIDs)
				}
			}

			grip.Info(message.Fields{
				"count":      count,
				"errs":       catcher.Len(),
				"collection": collName,
				"database":   dbName,
				"batches":    batches,
			})

			return catcher.Resolve()
		},
	}
}

func fromMDB() cli.Command {
	return cli.Command{
		Name:  "mongodb",
		Usage: "dump data from a mongodb collection into a FDTC file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  output,
				Usage: "write FTDC data in JSON format to this file (default: stdout)",
			},

			cli.StringFlag{
				Name:  "url",
				Usage: "specify the mongodb url",
				Value: "mongodb://127.0.0.1:27017",
			},
			cli.StringFlag{
				Name:  "collection, coll, c",
				Usage: "specify the name of the collection to load data too",
				Value: fmt.Sprintf("ftdc.%d", time.Now().Unix()),
			},
			cli.StringFlag{
				Name:  "database, db, d",
				Value: "curator",
			},
			cli.IntFlag{
				Name:  "batchSize",
				Value: 1000,
			},
			cli.BoolFlag{
				Name:  "continue",
				Usage: "continue on error during insertion",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			outfn := c.String(output)
			dbName := c.String("database")
			mdburl := c.String("url")
			collName := c.String("collection")
			batchSize := c.Int("batchSize")
			shouldContinue := c.Bool("continue")

			client, err := mongo.NewClient(options.Client().ApplyURI(mdburl))
			if err != nil {
				return errors.Wrap(err, "problem creating mongodb client")
			}

			connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err = client.Connect(connCtx); err != nil {
				return errors.Wrap(err, "problem connecting to mongodb")
			}

			coll := client.Database(dbName).Collection(collName)
			size, err := coll.CountDocuments(ctx, struct{}{})
			if err != nil {
				return errors.Wrap(err, "problem finding number of source documents")
			}
			if size == 0 {
				return errors.New("cannot write data from collection without documents")
			}

			var out *os.File
			if outfn == "" {
				out = os.Stdout
			} else {
				if _, err = os.Stat(outfn); !os.IsNotExist(err) {
					return errors.Errorf("cannot export collection to %s, file already exists", outfn)
				}

				out, err = os.Create(outfn)
				if err != nil {
					return errors.Wrapf(err, "problem opening file '%s'", outfn)
				}
				defer func() { grip.EmergencyFatal(out.Close()) }()
			}

			cursor, err := coll.Find(ctx, birch.NewDocument())
			if err != nil {
				return errors.Wrap(err, "problem finding documents")
			}
			defer func() { grip.Error(cursor.Close(ctx)) }()

			collector := ftdc.NewStreamingDynamicCollector(batchSize, out)
			defer func() { grip.EmergencyFatal(ftdc.FlushCollector(collector, out)) }()
			catcher := grip.NewBasicCatcher()
			count := 0

			for cursor.Next(ctx) {
				doc := birch.NewDocument()
				err := cursor.Decode(doc)
				catcher.Add(err)
				if err != nil {
					if !shouldContinue {
						break
					}
					continue
				}

				err = collector.Add(doc)
				catcher.Add(err)
				if err != nil {
					if !shouldContinue {
						break
					}
					continue
				}

				count++
			}

			catcher.Add(cursor.Err())
			catcher.Add(ftdc.FlushCollector(collector, out))
			grip.Info(message.Fields{
				"count":      count,
				"size":       size,
				"errs":       catcher.Len(),
				"collection": collName,
				"database":   dbName,
				"file":       outfn,
			})

			return catcher.Resolve()
		},
	}
}

func toT2() cli.Command {
	return cli.Command{
		Name:  "t2",
		Usage: "write data from genny output file to t2 compatible FTDC",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  input,
				Usage: "source genny output data file",
			},
			cli.StringFlag{
				Name:  output,
				Usage: "write genny output data in FTDC format `FILE` (default: stdout)",
			},
		},
		Before: requireFileExists(input, false),
		Action: func(c *cli.Context) error {
			inputPath := c.String(input)
			outputPath := c.String(output)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			inputFile, err := os.Open(inputPath)
			if err != nil {
				return errors.Wrapf(err, "problem opening file '%s'", inputPath)
			}
			defer func() { grip.Warning(inputFile.Close()) }()

			// All genny output files are named using the workload actor and operation.
			//   i.e., Actor.Operation.ftdc
			//
			// We get the actor and operation names from the ftdc filepath for use
			// in the translation process.
			fileName := filepath.Base(inputPath)
			actorOperation := strings.Split(fileName, ".ftdc")

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
				defer func() { grip.Warning(outputFile.Close()) }()
			}

			if err := ftdc.TranslateGenny(ctx, ftdc.ReadChunks(ctx, inputFile), outputFile, actorOperation[0]); err != nil {
				return errors.Wrap(err, "problem parsing ftdc")
			}
			return nil
		},
	}
}
