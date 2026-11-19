package sequencerrecovery

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	conductorrpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

const (
	configFlagName        = "config"
	networkFlagName       = "network"
	supernodeRPCFlagName  = "supernode-rpc"
	certFlagName          = "cert"
	stallAfterFlagName    = "stall-after"
	executeFlagName       = "execute"
	timeoutFlagName       = "timeout"
	pollIntervalFlagName  = "poll-interval"
	advanceBlocksFlagName = "advance-blocks"
	caughtUpFlagName      = "caught-up-within"
)

func commonFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     configFlagName,
			Usage:    "op-conductor-ops config file (netchef renders it to scripts/op-conductor-ops/<chain>.toml).",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "CONFIG"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     networkFlagName,
			Usage:    "Network (chain) name in the config file.",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "NETWORK"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     supernodeRPCFlagName,
			Usage:    "Rollup RPC of the supernode for this chain, i.e. the follow source the sequencers use (the chain's proxyd-cl).",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "SUPERNODE_RPC"),
			Required: true,
		},
		&cli.StringFlag{
			Name:    certFlagName,
			Usage:   "CA bundle for the RPC endpoints. Overrides cert_path from the config file.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "CERT"),
		},
		&cli.DurationFlag{
			Name:    stallAfterFlagName,
			Usage:   "Count the chain as stalled when the newest unsafe head is older than this.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "STALL_AFTER"),
			Value:   30 * time.Second,
		},
	}
}

// Subcommands returns the status and recover commands.
func Subcommands(envPrefix string) []*cli.Command {
	return []*cli.Command{
		{
			Name:  "status",
			Usage: "report the health of the conductor cluster and each sequencer; exits 1 when unhealthy",
			Flags: commonFlags(envPrefix),
			Action: func(cliCtx *cli.Context) error {
				return withCluster(cliCtx, func(ctx context.Context, _ log.Logger, c *Cluster) error {
					st, err := c.Status(ctx)
					if err != nil {
						return err
					}
					printStatus(cliCtx.App.Writer, st)
					if len(st.SequencingProblems()) > 0 || len(st.MemberProblems()) > 0 {
						return cli.Exit("unhealthy", 1)
					}
					return nil
				})
			},
		},
		{
			Name:  "recover",
			Usage: "restart sequencing on the canonical chain after a reorg; prints the plan unless --execute is set",
			Flags: append(commonFlags(envPrefix),
				&cli.BoolFlag{
					Name:    executeFlagName,
					Usage:   "Run the plan. Without it, only print the plan.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "EXECUTE"),
				},
				&cli.DurationFlag{
					Name:    timeoutFlagName,
					Usage:   "Maximum time for each wait in the plan.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "TIMEOUT"),
					Value:   2 * time.Minute,
				},
				&cli.DurationFlag{
					Name:    pollIntervalFlagName,
					Usage:   "How often waits re-read the cluster.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "POLL_INTERVAL"),
					Value:   time.Second,
				},
				&cli.DurationFlag{
					Name:    caughtUpFlagName,
					Usage:   "How close to wall-clock time the restarted sequencer's head must be before the conductors resume.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "CAUGHT_UP_WITHIN"),
					Value:   10 * time.Second,
				},
				&cli.Uint64Flag{
					Name:    advanceBlocksFlagName,
					Usage:   "Blocks the restarted sequencer must build before the conductors resume.",
					EnvVars: opservice.PrefixEnvVar(envPrefix, "ADVANCE_BLOCKS"),
					Value:   2,
				},
			),
			Action: func(cliCtx *cli.Context) error {
				return withCluster(cliCtx, func(ctx context.Context, lgr log.Logger, c *Cluster) error {
					st, err := c.Status(ctx)
					if err != nil {
						return err
					}
					printStatus(cliCtx.App.Writer, st)
					steps, err := c.Plan(st, RecoverConfig{
						PollInterval:   cliCtx.Duration(pollIntervalFlagName),
						Timeout:        cliCtx.Duration(timeoutFlagName),
						AdvanceBlocks:  cliCtx.Uint64(advanceBlocksFlagName),
						CaughtUpWithin: cliCtx.Duration(caughtUpFlagName),
					})
					if err != nil {
						return err
					}
					if len(steps) == 0 {
						fmt.Fprintln(cliCtx.App.Writer, "Sequencing is healthy; nothing to recover.")
						return nil
					}
					fmt.Fprintln(cliCtx.App.Writer, "Plan:")
					for i, s := range steps {
						fmt.Fprintf(cliCtx.App.Writer, "  %d. %s\n", i+1, s.Desc)
					}
					if !cliCtx.Bool(executeFlagName) {
						fmt.Fprintln(cliCtx.App.Writer, "Dry run. Re-run with --execute to apply.")
						return nil
					}
					return Run(ctx, lgr, steps)
				})
			},
		},
	}
}

func withCluster(cliCtx *cli.Context, fn func(ctx context.Context, lgr log.Logger, c *Cluster) error) error {
	ctx := cliCtx.Context
	lgr := log.NewLogger(logcli.NewLogHandler(cliCtx.App.ErrWriter, logcli.DefaultCLIConfig())).WithContext(ctx)

	cfg, err := LoadConfig(cliCtx.String(configFlagName), cliCtx.String(networkFlagName))
	if err != nil {
		return err
	}
	certPath := cfg.CertPath
	if p := cliCtx.String(certFlagName); p != "" {
		certPath = p
	}
	httpClient, err := newHTTPClient(certPath)
	if err != nil {
		return err
	}
	dial := func(url string) (opclient.RPC, error) {
		return opclient.NewRPC(ctx, lgr, url, opclient.WithGethRPCOptions(gethrpc.WithHTTPClient(httpClient)))
	}

	snRPC, err := dial(cliCtx.String(supernodeRPCFlagName))
	if err != nil {
		return fmt.Errorf("dial supernode: %w", err)
	}
	defer snRPC.Close()
	c := &Cluster{
		Supernode:  sources.NewRollupClient(snRPC),
		StallAfter: cliCtx.Duration(stallAfterFlagName),
		Now:        time.Now,
	}
	for _, s := range cfg.Sequencers {
		condRPC, err := dial(s.ConductorURL)
		if err != nil {
			return fmt.Errorf("dial conductor %s: %w", s.ID, err)
		}
		defer condRPC.Close()
		nodeRPC, err := dial(s.NodeURL)
		if err != nil {
			return fmt.Errorf("dial op-node %s: %w", s.ID, err)
		}
		defer nodeRPC.Close()
		c.Members = append(c.Members, Member{
			Sequencer: s,
			Conductor: conductorrpc.NewAPIClient(condRPC),
			Node:      sources.NewRollupClient(nodeRPC),
		})
	}
	return fn(ctx, lgr, c)
}

func newHTTPClient(certPath string) (*http.Client, error) {
	if certPath == "" {
		return http.DefaultClient, nil
	}
	pem, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read CA bundle: %w", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no certificates in %s", certPath)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	return &http.Client{Transport: transport}, nil
}

func printStatus(w io.Writer, st *Status) {
	fmt.Fprintf(w, "supernode: unsafe=%d safe=%d (%s) finalized=%d\n",
		st.Supernode.UnsafeL2.Number, st.Supernode.SafeL2.Number, st.Supernode.SafeL2.Hash, st.Supernode.FinalizedL2.Number)
	fmt.Fprintf(w, "raft leader: %s\n", orNone(st.LeaderID))
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SEQUENCER\tLEADER\tPAUSED\tHEALTHY\tACTIVE\tUNSAFE\tSAFE\tCANONICAL\tERROR")
	for _, m := range st.Members {
		var unsafe, safe string
		if m.Sync != nil {
			unsafe = fmt.Sprintf("%d", m.Sync.UnsafeL2.Number)
			safe = fmt.Sprintf("%d", m.Sync.SafeL2.Number)
		}
		var errStr string
		if m.Err != nil {
			errStr = m.Err.Error()
		}
		fmt.Fprintf(tw, "%s\t%t\t%t\t%t\t%t\t%s\t%s\t%t\t%s\n",
			m.ID, m.Leader, m.Paused, m.SequencerHealthy, m.SequencerActive, unsafe, safe, m.Canonical, errStr)
	}
	_ = tw.Flush()
	seq, members := st.SequencingProblems(), st.MemberProblems()
	if len(seq) == 0 && len(members) == 0 {
		fmt.Fprintln(w, "healthy")
		return
	}
	if len(seq) > 0 {
		fmt.Fprintf(w, "sequencing problems (recover fixes these):\n  %s\n", strings.Join(seq, "\n  "))
	}
	if len(members) > 0 {
		fmt.Fprintf(w, "member problems (not fixed by recover; resync these nodes):\n  %s\n", strings.Join(members, "\n  "))
	}
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
