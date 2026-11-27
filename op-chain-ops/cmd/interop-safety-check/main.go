// Command interop-safety-check independently re-validates every executing message in
// the cross-safe range of each op-supernode, checks that all supernodes agree on the
// cross-safe chain, and reports cross-safe progress. See README.md.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/interopsafety"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
)

var (
	Version     = "v0.0.0"
	VersionMeta = "dev"
	GitCommit   string
	GitDate     string
)

const envPrefix = "INTEROP_SAFETY_CHECK"

var (
	chainsFlag = &cli.Uint64SliceFlag{
		Name:     "chains",
		Usage:    "Chain IDs of the dependency set.",
		EnvVars:  opservice.PrefixEnvVar(envPrefix, "CHAINS"),
		Required: true,
	}
	fromFlag = &cli.StringSliceFlag{
		Name:    "from",
		Usage:   "First block to check per chain, as <chainID>:<block> (normally the Lagoon activation block).",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "FROM"),
	}
	supernodesFlag = &cli.StringSliceFlag{
		Name:     "supernode",
		Usage:    "Supernode as <name>=<rollup RPC base URL>; the per-chain RPC is <base>/<chainID>.",
		EnvVars:  opservice.PrefixEnvVar(envPrefix, "SUPERNODES"),
		Required: true,
	}
	elsFlag = &cli.StringSliceFlag{
		Name:     "el",
		Usage:    "EL used by a supernode, as <name>:<chainID>=<EL RPC URL>. Every supernode needs one per chain.",
		EnvVars:  opservice.PrefixEnvVar(envPrefix, "ELS"),
		Required: true,
	}
	goneTxsFlag = &cli.StringSliceFlag{
		Name:    "gone-tx",
		Usage:   "Invalidated tx hash that must not be canonical on any chain.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "GONE_TXS"),
	}
	depositOnlyFlag = &cli.StringSliceFlag{
		Name:    "deposit-only",
		Usage:   "Replaced block that must hold only deposits, as <chainID>:<block>.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "DEPOSIT_ONLY"),
	}
	intervalFlag = &cli.DurationFlag{
		Name:    "interval",
		Usage:   "Repeat every interval; 0 runs one pass. Later passes only check newly cross-safe blocks.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "INTERVAL"),
	}
	stallAfterFlag = &cli.DurationFlag{
		Name:    "stall-after",
		Usage:   "Report a supernode whose cross-safe head has not advanced for this long.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "STALL_AFTER"),
		Value:   15 * time.Minute,
	}
	outFlag = &cli.StringFlag{
		Name:    "out",
		Usage:   "Append one JSON line per pass to this file.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "OUT"),
	}
	rpcTimeoutFlag = &cli.DurationFlag{
		Name:    "rpc-timeout",
		Usage:   "Timeout of each RPC call.",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "RPC_TIMEOUT"),
		Value:   20 * time.Second,
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "interop-safety-check"
	app.Usage = "independently re-validate every executing message in the op-supernodes' cross-safe chains"
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, VersionMeta)
	app.Flags = cliapp.ProtectFlags(append([]cli.Flag{chainsFlag, fromFlag, supernodesFlag, elsFlag, goneTxsFlag, depositOnlyFlag,
		intervalFlag, stallAfterFlag, outFlag, rpcTimeoutFlag}, logcli.CLIFlags(envPrefix)...))
	app.Action = run
	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func splitPair(s, sep string) (string, string, error) {
	a, b, ok := strings.Cut(s, sep)
	if !ok || a == "" || b == "" {
		return "", "", fmt.Errorf("%q: expected <a>%s<b>", s, sep)
	}
	return a, b, nil
}

func chainBlock(s string) (uint64, uint64, error) {
	a, b, err := splitPair(s, ":")
	if err != nil {
		return 0, 0, err
	}
	c, err := strconv.ParseUint(a, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%q: chain ID: %w", s, err)
	}
	n, err := strconv.ParseUint(b, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%q: block: %w", s, err)
	}
	return c, n, nil
}

func run(c *cli.Context) error {
	ctx := c.Context
	lgr := logcli.NewLogger(c.App.ErrWriter, logcli.ReadCLIConfig(c))
	chains := c.Uint64Slice(chainsFlag.Name)
	timeout := c.Duration(rpcTimeoutFlag.Name)

	cfg := interopsafety.Config{From: map[uint64]uint64{}, DepositOnly: map[uint64][]uint64{}, StallAfter: c.Duration(stallAfterFlag.Name)}
	for _, s := range c.StringSlice(fromFlag.Name) {
		chain, n, err := chainBlock(s)
		if err != nil {
			return fmt.Errorf("--from: %w", err)
		}
		cfg.From[chain] = n
	}
	for _, s := range c.StringSlice(depositOnlyFlag.Name) {
		chain, n, err := chainBlock(s)
		if err != nil {
			return fmt.Errorf("--deposit-only: %w", err)
		}
		cfg.DepositOnly[chain] = append(cfg.DepositOnly[chain], n)
	}
	for _, s := range c.StringSlice(goneTxsFlag.Name) {
		cfg.GoneTxs = append(cfg.GoneTxs, common.HexToHash(s))
	}

	byName := map[string]*interopsafety.Supernode{}
	var supernodes []*interopsafety.Supernode
	rollups := map[string]map[uint64]*gethrpc.Client{}
	for _, s := range c.StringSlice(supernodesFlag.Name) {
		name, base, err := splitPair(s, "=")
		if err != nil {
			return fmt.Errorf("--supernode: %w", err)
		}
		sn := &interopsafety.Supernode{Name: name, Chains: map[uint64]interopsafety.ChainReader{}}
		rollups[name] = map[uint64]*gethrpc.Client{}
		for _, chain := range chains {
			cl, err := gethrpc.DialContext(ctx, fmt.Sprintf("%s/%d", strings.TrimSuffix(base, "/"), chain))
			if err != nil {
				return fmt.Errorf("dial supernode %s chain %d: %w", name, chain, err)
			}
			defer cl.Close()
			rollups[name][chain] = cl
		}
		byName[name] = sn
		supernodes = append(supernodes, sn)
	}
	for _, s := range c.StringSlice(elsFlag.Name) {
		lhs, url, err := splitPair(s, "=")
		if err != nil {
			return fmt.Errorf("--el: %w", err)
		}
		name, chainStr, err := splitPair(lhs, ":")
		if err != nil {
			return fmt.Errorf("--el: %w", err)
		}
		chain, err := strconv.ParseUint(chainStr, 10, 64)
		if err != nil {
			return fmt.Errorf("--el %q: chain ID: %w", s, err)
		}
		sn, ok := byName[name]
		if !ok {
			return fmt.Errorf("--el %q: unknown supernode %q", s, name)
		}
		el, err := gethrpc.DialContext(ctx, url)
		if err != nil {
			return fmt.Errorf("dial EL %s: %w", s, err)
		}
		defer el.Close()
		sn.Chains[chain] = &interopsafety.RPCReader{Rollup: rollups[name][chain], EL: el, Timeout: timeout}
	}
	for _, sn := range supernodes {
		for _, chain := range chains {
			if _, ok := sn.Chains[chain]; !ok {
				return fmt.Errorf("supernode %s has no --el for chain %d", sn.Name, chain)
			}
		}
	}

	k := interopsafety.NewChecker(lgr, cfg, chains, supernodes, time.Now)
	for {
		k.Pass(ctx)
		agreement := k.Agreement(ctx)
		k.CheckReplacements(ctx)
		stalled := k.Stalled()
		for _, s := range stalled {
			lgr.Error("Cross-safe stalled", "supernode/chain", s, "stall_after", cfg.StallAfter)
		}
		lgr.Info("Pass complete", "failures", len(k.Failures()), "agreement", agreement, "stalled", stalled)
		if out := c.String(outFlag.Name); out != "" {
			if err := appendJSONLine(out, map[string]any{"t": time.Now().Unix(), "failures": len(k.Failures()), "agreement": agreement, "stalled": stalled, "state": k.State()}); err != nil {
				lgr.Warn("Failed to write --out", "err", err)
			}
		}
		interval := c.Duration(intervalFlag.Name)
		if interval == 0 {
			if len(k.Failures()) > 0 {
				return fmt.Errorf("%d safety failures", len(k.Failures()))
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

func appendJSONLine(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
