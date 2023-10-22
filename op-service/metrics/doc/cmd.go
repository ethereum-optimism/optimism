package doc

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type Metrics interface {
	Document() []metrics.DocumentedMetric
}

func NewSubcommands(m Metrics) cli.Commands {
	return cli.Commands{
		{
			Name:  "metrics",
			Usage: "Dumps a list of supported metrics to stdout",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "format",
					Value: "markdown",
					Usage: "Output format (json|markdown)",
				},
			},
			Action: func(ctx *cli.Context) error {
				supportedMetrics := m.Document()
				format := ctx.String("format")

				if format != "markdown" && format != "json" {
					return fmt.Errorf("invalid format: %s", format)
				}

				if format == "json" {
					enc := json.NewEncoder(os.Stdout)
					return enc.Encode(supportedMetrics)
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
				table.SetCenterSeparator("|")
				table.SetAutoWrapText(false)
				table.SetHeader([]string{"Metric", "Description", "Labels", "Type"})
				var data [][]string
				for _, metric := range supportedMetrics {
					labels := strings.Join(metric.Labels, ",")
					data = append(data, []string{metric.Name, metric.Help, labels, metric.Type})
				}
				table.AppendBulk(data)
				table.Render()
				return nil
			},
		},
	}
}
