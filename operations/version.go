package operations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mongodb/curator"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type versionInfo struct {
	Curator      string `json:"curator"`
	Jasper       string `json:"jasper_proto"`
	CedarMetrics string `json:"cedar_metrics_proto"`
}

func (v versionInfo) String() string {
	return strings.Join([]string{
		"Curator Version Info:",
		"\n\t", "Build: ", v.Curator,
		"\n\t", "Jasper: ", v.Jasper,
		"\n\t", "CedarMetrics: ", v.CedarMetrics,
	}, "")
}

// Version command line function.
func Version() cli.Command {
	return cli.Command{
		Name:  "version",
		Usage: "prints version information",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "json",
				Usage: "specify this option to output data as JSON",
			},
		},
		Action: func(c *cli.Context) error {
			isJSON := c.Bool("json")

			info := versionInfo{
				Curator:      curator.BuildRevision,
				Jasper:       curator.JasperChecksum,
				CedarMetrics: curator.CedarMetricsChecksum,
			}
			if isJSON {
				out, err := json.MarshalIndent(info, "", "   ")
				if err != nil {
					return errors.Wrap(err, "marshalling JSON")
				}
				fmt.Println(string(out))
				return nil
			}

			fmt.Println(info)
			return nil
		},
	}
}
