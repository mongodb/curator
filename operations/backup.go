package operations

import (
	"context"
	"time"

	"github.com/evergreen-ci/pail"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/anser/backup"
	"github.com/mongodb/anser/model"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Backup provides a commandline tool for backing up mongodb
// collections to s3.
func Backup() cli.Command {
	return cli.Command{
		Name:  "backup",
		Usage: "writes backups of mongodb collections directly to s3",
		Flags: baseS3Flags(
			cli.StringFlag{
				Name:  "prefix",
				Usage: "a prefix for keys",
			},
			cli.StringFlag{
				Name:  "permissions",
				Usage: "canned ACL to apply to the files",
				Value: string(pail.S3PermissionsPrivate),
			},
			cli.StringFlag{
				Name:  "mongodbURI, mdb",
				Value: "mongodb://localhost:27017",
				Usage: "specify mongodb server UI",
			},
			cli.StringFlag{
				Name:  "database, db, d",
				Usage: "specify a database name",
			},
			cli.StringSliceFlag{
				Name:  "collection, c",
				Usage: "specify a collection name",
			},
			cli.BoolFlag{
				Name:  "gzip, z",
				Usage: "specify to enable compression of artifacts",
			},
		),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			colls := c.StringSlice("collection")
			if len(colls) == 0 {
				return errors.New("must specify one or more collection")
			}
			dbName := c.String("database")
			if dbName == "" {
				return errors.New("must specify a database name")
			}

			startAt := time.Now()

			client, err := mongo.NewClient(options.Client().ApplyURI(c.String("mongodbURI")))
			if err != nil {
				return errors.Wrap(err, "constructing client")
			}

			if err = client.Connect(ctx); err != nil {
				return errors.Wrap(err, "connecting client")
			}
			httpClient := utility.GetHTTPClient()
			defer utility.PutHTTPClient(httpClient)
			bucket, err := pail.NewS3BucketWithHTTPClient(ctx, httpClient,
				pail.S3Options{
					SharedCredentialsProfile: c.String("profile"),
					Region:                   c.String("region"),
					Name:                     c.String("bucket"),
					Permissions:              pail.S3Permissions(c.String("permissions")),
					Verbose:                  c.Bool("verbose"),
					Prefix:                   c.String("prefix"),
					Compress:                 c.Bool("gzip"),
				})
			if err != nil {
				return errors.Wrap(err, "constructing bucket client")
			}

			seen := 0
			catcher := grip.NewBasicCatcher()
			for _, coll := range colls {
				seen++
				collStartAt := time.Now()
				opts := backup.Options{
					NS: model.Namespace{
						DB:         dbName,
						Collection: coll,
					},
					Target:        bucket.Writer,
					EnableLogging: true,
				}
				err := backup.Collection(ctx, client, opts)
				msg := message.Fields{
					"ns":        opts.NS.String(),
					"total":     len(colls),
					"completed": seen,
					"dur_secs":  time.Since(collStartAt).Seconds(),
				}
				catcher.Add(err)
				grip.InfoWhen(err == nil, msg)
				grip.Error(message.WrapError(err, msg))
			}

			msg := message.Fields{
				"db":          dbName,
				"collections": colls,
				"dur_secs":    time.Since(startAt).Seconds(),
				"target":      c.String("bucket"),
				"prefix":      c.String("prefix"),
			}

			grip.InfoWhen(!catcher.HasErrors(), msg)
			grip.Error(message.WrapError(catcher.Resolve(), msg))

			return catcher.Resolve()
		},
	}
}
