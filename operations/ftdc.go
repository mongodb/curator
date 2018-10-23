package operations

import (
	"context"
	"fmt"
	"os"

	"github.com/mongodb/ftdc"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func FTDC() cli.Command {
	return cli.Command{
		Name:  "ftdc",
		Usage: "tools for running FTDC parsing and generating tools",
		Flags: []cli.Flag{},
		Subcommands: []cli.Command{
			ftdcDump(),
		},
	}
}

func ftdcDump() cli.Command {
	return cli.Command{
		Name:  "dump",
		Usage: "dump ftdc data from a given BSON file",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "path",
				Usage: "dump FTDC data from this file",
			},
			cli.BoolFlag{
				Name:  "structured",
				Usage: "dump FTDC data in a structured format",
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			path := c.String("path")
			if path == "" {
				return errors.New("path is not specified")
			}

			f, err := os.Open(path)
			if err != nil {
				return errors.Wrapf(err, "problem opening file '%s'", path)
			}
			defer f.Close()

			var iter ftdc.Iterator
			if c.Bool("structured") {
				iter = ftdc.ReadStructuredMetrics(ctx, f)
			} else {
				iter = ftdc.ReadMetrics(ctx, f)
			}

			for iter.Next(ctx) {
				fmt.Println(iter.Document().ToExtJSON(false))
			}

			return iter.Err()
		},
	}
}
