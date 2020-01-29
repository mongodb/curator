package operations

import (
	"context"

	"github.com/evergreen-ci/pail"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/anser/backup"
	"github.com/mongodb/anser/model"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
			cli.StringFlag{
				Name:  "collection, c",
				Usage: "specify a collection name",
			},
		),
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := mongo.NewClient(options.Client().ApplyURI(c.String("mongodbURI")))
			if err != nil {
				return errors.Wrap(err, "problem constructing client")
			}

			if err = client.Connect(ctx); err != nil {
				return errors.Wrap(err, "problem connecting client")
			}
			httpClient := utility.GetHTTPClient()
			defer utility.PutHTTPClient(httpClient)
			bucket, err := pail.NewS3BucketWithHTTPClient(httpClient,
				pail.S3Options{
					SharedCredentialsProfile: c.String("profile"),
					Region:                   c.String("region"),
					Name:                     c.String("bucket"),
					Permissions:              pail.S3Permissions(c.String("permissions")),
					Verbose:                  c.Bool("verbose"),
					Prefix:                   c.String("prefix"),
				})
			if err != nil {
				return errors.Wrap(err, "problem client")
			}

			opts := backup.Options{
				NS: model.Namespace{
					DB:         c.String("database"),
					Collection: c.String("collection"),
				},
				Target:        bucket.Writer,
				EnableLogging: true,
			}

			return backup.Collection(ctx, client, opts)
		},
	}
}
