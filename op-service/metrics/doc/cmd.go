package doc

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

				writeMarkdownTableHeader("Metric", "Description", "Labels", "Type")
				for _, metric := range supportedMetrics {
					labels := strings.Join(metric.Labels, ",")
					writeMarkdownTableRow(metric.Name, metric.Help, labels, metric.Type)
				}
				return nil
			},
		},
	}
}

func writeMarkdownTableHeader(columns ...string) {
	writeMarkdownTableRow(columns...)
	for i := range columns {
		if i > 0 {
			fmt.Fprint(os.Stdout, "|")
		}
		fmt.Fprint(os.Stdout, "---")
	}
	fmt.Fprintln(os.Stdout)
}

func writeMarkdownTableRow(columns ...string) {
	for i, column := range columns {
		if i > 0 {
			fmt.Fprint(os.Stdout, "|")
		}
		fmt.Fprintf(os.Stdout, " %s ", markdownCell(column))
	}
	fmt.Fprintln(os.Stdout)
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return value
}
