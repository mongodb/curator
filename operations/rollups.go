package operations

import (
	"context"
	"os"

	"github.com/evergreen-ci/utility"
	"github.com/mongodb/curator/operations/rollups"
	"github.com/mongodb/ftdc"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	ftdcFileLocation = "inputFile"
	rollupOutputFile = "outputFile"
)

type CedarInputData struct {
	Name          string
	Value         interface{}
	Type          string
	UserSubmitted bool
}

func CalculateRollups() cli.Command {
	return cli.Command{
		Name:  "calculate-rollups",
		Usage: "Create a rollup of performance data from a FTDC file",
		Flags: []cli.Flag{cli.StringFlag{
			Name:  ftdcFileLocation,
			Usage: "Specify where the FTDC file to calculate is located",
		}, cli.StringFlag{
			Name:  rollupOutputFile,
			Usage: "Specify where to output the rollup file",
		}},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fileName := c.String(ftdcFileLocation)
			outputFileName := c.String(rollupOutputFile)

			fileData, err := os.Open(fileName)
			if err != nil {
				return errors.WithStack(err)
			}
			ftdcData := ftdc.ReadChunks(ctx, fileData)

			rollupData, err := rollups.CalculateDefaultRollups(ftdcData, false)
			if err != nil {
				return errors.WithStack(err)
			}
			formattedRollupData := []CedarInputData{}
			for _, data := range rollupData {
				formattedRollupData = append(formattedRollupData, CedarInputData{
					Name:          data.Name,
					Value:         data.Value,
					Type:          string(data.MetricType),
					UserSubmitted: data.UserSubmitted,
				})
			}

			return errors.WithStack(utility.WriteJSONFile(outputFileName, formattedRollupData))
		},
	}
}
