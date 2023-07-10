package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/olekukonko/tablewriter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"
)

// TODO: Provide info metric with version automatically

type Service struct {
	registerMetrics RegisterMetrics
	config          CLIConfig
	registry        *prometheus.Registry
	factory         Factory
}

func (s *Service) Flags(envVarPrefix string) []cli.Flag {
	return CLIFlags(envVarPrefix)
}

func (s *Service) Subcommands() cli.Commands {
	return cli.Commands{
		{
			Name: "doc",
			Subcommands: cli.Commands{
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
						s.registerMetrics(s)
						supportedMetrics := s.Document()
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
			},
		},
	}
}

func (s *Service) Init(logger log.Logger, ctx *cli.Context) error {
	s.config = ReadCLIConfig(ctx)
	if err := s.config.Check(); err != nil {
		return fmt.Errorf("metrics config error: %w", err)
	}
	if s.config.Enabled {
		logger.Info("starting metrics server", "addr", s.config.ListenAddr, "port", s.config.ListenPort)
		go func() {
			if err := ListenAndServe(ctx.Context, s.registry, s.config.ListenAddr, s.config.ListenPort); err != nil {
				logger.Error("error starting metrics server", err)
			}
		}()
		// TODO: Support starting balance metrics?
	}
	return nil
}

func (s *Service) StartBalanceMetrics(ctx context.Context, l log.Logger, ns string, client *ethclient.Client, account common.Address) {
	if !s.config.Enabled {
		return
	}
	LaunchBalanceMetrics(ctx, l, s.registry, ns, client, account)
}

func (s *Service) Factory() Factory {
	return s.factory
}

func (s *Service) Document() []DocumentedMetric {
	return s.factory.Document()
}

type RegisterMetrics func(service *Service)

func NewService(registerMetrics RegisterMetrics) *Service {
	registry := NewRegistry()
	return &Service{
		registerMetrics: registerMetrics,
		registry:        registry,
		factory:         With(registry),
	}
}
