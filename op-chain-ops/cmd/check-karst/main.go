package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-karst/karsttest"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

var (
	prefix     = "CHECK_KARST"
	EndpointL2 = &cli.StringFlag{
		Name:    "l2",
		Usage:   "L2 execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2"),
		Value:   "http://localhost:9545",
	}
	AccountKey = &cli.StringFlag{
		Name:     "account",
		Usage:    "Hex-encoded private key of a funded test account used to send check txs",
		EnvVars:  op_service.PrefixEnvVar(prefix, "ACCOUNT"),
		Required: true,
	}
	EndpointL1 = &cli.StringFlag{
		Name:     "l1",
		Usage:    "L1 execution RPC endpoint",
		EnvVars:  op_service.PrefixEnvVar(prefix, "L1"),
		Required: true,
	}
	L1AccountKey = &cli.StringFlag{
		Name:     "l1-account",
		Usage:    "Hex-encoded private key of a funded L1 test account used to submit the deposit",
		EnvVars:  op_service.PrefixEnvVar(prefix, "L1_ACCOUNT"),
		Required: true,
	}
	PortalAddress = &cli.StringFlag{
		Name:     "portal",
		Usage:    "L1 OptimismPortal contract address",
		EnvVars:  op_service.PrefixEnvVar(prefix, "PORTAL"),
		Required: true,
	}
	FlashblocksWS = &cli.StringFlag{
		Name: "flashblocks-ws",
		Usage: "Optional flashblocks websocket endpoint (any flashblocks-API stream, e.g. op-rbuilder or rollup-boost). " +
			"When set, every L2 tx a check includes on-chain must also appear in a flashblock for its inclusion block.",
		EnvVars: op_service.PrefixEnvVar(prefix, "FLASHBLOCKS_WS"),
	}
	SpamAccounts = &cli.Uint64Flag{
		Name:    "accounts",
		Usage:   "Number of ephemeral EOAs to fund and round-robin spam from (reth caps in-flight txs per account at 16, so spamming needs many accounts)",
		EnvVars: op_service.PrefixEnvVar(prefix, "ACCOUNTS"),
		Value:   50,
	}
	SpamRPS = &cli.Uint64Flag{
		Name:    "rps",
		Usage:   "Base send rate per block slot for the burst scheduler (ramps up while accepted, backs off on errors)",
		EnvVars: op_service.PrefixEnvVar(prefix, "RPS"),
		Value:   50,
	}
	SpamFundGwei = &cli.Uint64Flag{
		Name:    "fund-gwei",
		Usage:   "Whole gwei transferred from --account to each spam EOA up front",
		EnvVars: op_service.PrefixEnvVar(prefix, "FUND_GWEI"),
		Value:   10,
	}
	SpamBlockTime = &cli.DurationFlag{
		Name:    "block-time",
		Usage:   "L2 block time, used to pace the burst scheduler and the reliable-inclusion retry loop",
		EnvVars: op_service.PrefixEnvVar(prefix, "BLOCK_TIME"),
		Value:   2 * time.Second,
	}
)

// flashblockVerifyTimeout bounds how long, after a check's txs have landed
// on-chain, we wait for the matching flashblocks to be observed on the stream.
// The flashblock for an inclusion block precedes sealing, so this only absorbs
// websocket propagation latency.
const flashblockVerifyTimeout = 30 * time.Second

func makeFlags() []cli.Flag {
	flags := []cli.Flag{EndpointL2, AccountKey, FlashblocksWS}
	return append(flags, oplog.CLIFlags(prefix)...)
}

// checkEnv bundles the resolved per-invocation inputs that every subcommand
// needs.
type checkEnv struct {
	ctx      context.Context
	logger   log.Logger
	l2       *ethclient.Client
	key      *ecdsa.PrivateKey
	basePlan txplan.Option

	// fbTracker and fbStop are non-nil only when --flashblocks-ws is set.
	// fbTracker records and verifies flashblock inclusion of the L2 txs sent via
	// basePlan; fbStop tears down the websocket stream.
	fbTracker *karsttest.FlashblockTracker
	fbStop    func()
}

func (e *checkEnv) close() {
	if e.fbStop != nil {
		e.fbStop()
	}
	if e.l2 != nil {
		e.l2.Close()
	}
}

// verifyFlashblocks asserts every L2 tx a check included on-chain also appeared
// in a flashblock for its inclusion block. It is a no-op when --flashblocks-ws
// is unset. Call it after the check action and before close() (which stops the
// stream).
func (e *checkEnv) verifyFlashblocks() error {
	if e.fbTracker == nil {
		return nil
	}
	return e.fbTracker.Verify(e.ctx, flashblockVerifyTimeout)
}

func resolveEnv(c *cli.Context) (*checkEnv, error) {
	logCfg := oplog.ReadCLIConfig(c)
	logger := oplog.NewLogger(c.App.Writer, logCfg)

	c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
	l2Cl, err := ethclient.DialContext(c.Context, c.String(EndpointL2.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to dial L2 RPC: %w", err)
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(c.String(AccountKey.Name), "0x"))
	if err != nil {
		l2Cl.Close()
		return nil, fmt.Errorf("failed to parse account private key: %w", err)
	}

	env := &checkEnv{
		ctx:      c.Context,
		logger:   logger,
		l2:       l2Cl,
		key:      key,
		basePlan: karsttest.NewBasePlan(l2Cl, key),
	}

	if wsURL := c.String(FlashblocksWS.Name); wsURL != "" {
		tracker, trackOpt, stop, err := karsttest.StartFlashblockTracking(c.Context, logger, wsURL)
		if err != nil {
			l2Cl.Close()
			return nil, err
		}
		env.basePlan = txplan.Combine(env.basePlan, trackOpt)
		env.fbTracker = tracker
		env.fbStop = stop
	}

	return env, nil
}

// CheckAction is the shared signature for every karsttest check function the
// CLI exposes. CheckResult is discarded by the CLI; its block range is only
// useful to the acceptance test (for kona-host cross-checks).
type CheckAction func(ctx context.Context, logger log.Logger, l2 apis.EthCode, basePlan txplan.Option) error

func makeCommand(name string, fn CheckAction) *cli.Command {
	return &cli.Command{
		Name:  name,
		Flags: cliapp.ProtectFlags(makeFlags()),
		Action: func(c *cli.Context) error {
			env, err := resolveEnv(c)
			if err != nil {
				return err
			}
			defer env.close()
			if err := fn(env.ctx, env.logger, env.l2, env.basePlan); err != nil {
				return fmt.Errorf("command error: %w", err)
			}
			return env.verifyFlashblocks()
		},
	}
}

func makeAllCommand() *cli.Command {
	flags := append(makeFlags(), EndpointL1, L1AccountKey, PortalAddress)
	return &cli.Command{
		Name:  "all",
		Flags: cliapp.ProtectFlags(flags),
		Action: func(c *cli.Context) error {
			env, err := resolveEnv(c)
			if err != nil {
				return err
			}
			defer env.close()

			l1Cl, err := ethclient.DialContext(env.ctx, c.String(EndpointL1.Name))
			if err != nil {
				return fmt.Errorf("dial L1: %w", err)
			}
			defer l1Cl.Close()

			l1Key, err := crypto.HexToECDSA(strings.TrimPrefix(c.String(L1AccountKey.Name), "0x"))
			if err != nil {
				return fmt.Errorf("parse L1 account: %w", err)
			}

			portalHex := c.String(PortalAddress.Name)
			if !common.IsHexAddress(portalHex) {
				return fmt.Errorf("--portal must be a hex address, got %q", portalHex)
			}

			if err := karsttest.CheckAll(
				env.ctx, env.logger,
				&ethclientLatestBlockAdapter{env.l2},
				env.basePlan,
				common.HexToAddress(portalHex),
				crypto.PubkeyToAddress(l1Key.PublicKey),
				karsttest.NewBasePlan(l1Cl, l1Key),
				eth.OneHundredthEther,
			); err != nil {
				return fmt.Errorf("command error: %w", err)
			}
			return env.verifyFlashblocks()
		},
	}
}

// ethclientLatestBlockAdapter satisfies karsttest.LatestBlockFetcher by
// translating InfoAndTxsByLabel(Unsafe) → BlockByNumber(nil). Other labels
// aren't needed by the EIP-7934 check.
type ethclientLatestBlockAdapter struct{ *ethclient.Client }

func (a *ethclientLatestBlockAdapter) InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error) {
	if label != eth.Unsafe {
		return nil, nil, fmt.Errorf("unsupported block label %q (only %q is supported)", label, eth.Unsafe)
	}
	block, err := a.Client.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	return eth.BlockToInfo(block), block.Transactions(), nil
}

// makeBlockSizeCommand wires up the EIP-7934 block-size-disabled check.
// Unlike the EVM-level checks, it polls latest blocks for one whose tx data
// exceeds MaxBlockSize. Callers control the deadline via Ctrl+C
// (which cancels the interrupt-aware ctx).
func makeBlockSizeCommand() *cli.Command {
	return &cli.Command{
		Name:  "eip-7934",
		Flags: cliapp.ProtectFlags(makeFlags()),
		Action: func(c *cli.Context) error {
			env, err := resolveEnv(c)
			if err != nil {
				return err
			}
			defer env.close()
			if err := karsttest.CheckEIP7934BlockSizeDisabled(
				env.ctx, env.logger,
				&ethclientLatestBlockAdapter{env.l2},
				2*time.Second,
			); err != nil {
				return fmt.Errorf("command error: %w", err)
			}
			return env.verifyFlashblocks()
		},
	}
}

// makeDepositCommand wires up the deposit-bypass check, which (unlike the
// EVM-level checks) needs an L1 endpoint, an L1 funded account, and the L1
// portal address. These extra inputs are not part of the shared CheckAction
// signature, so the deposit command builds its own flag list and resolves L1
// state inline.
func makeDepositCommand() *cli.Command {
	flags := append(makeFlags(), EndpointL1, L1AccountKey, PortalAddress)
	return &cli.Command{
		Name:  "eip-7825-deposit",
		Flags: cliapp.ProtectFlags(flags),
		Action: func(c *cli.Context) error {
			env, err := resolveEnv(c)
			if err != nil {
				return err
			}
			defer env.close()

			l1Cl, err := ethclient.DialContext(env.ctx, c.String(EndpointL1.Name))
			if err != nil {
				return fmt.Errorf("dial L1: %w", err)
			}
			defer l1Cl.Close()

			l1Key, err := crypto.HexToECDSA(strings.TrimPrefix(c.String(L1AccountKey.Name), "0x"))
			if err != nil {
				return fmt.Errorf("parse L1 account: %w", err)
			}

			portalHex := c.String(PortalAddress.Name)
			if !common.IsHexAddress(portalHex) {
				return fmt.Errorf("--portal must be a hex address, got %q", portalHex)
			}

			if _, err := karsttest.CheckEIP7825DepositBypass(
				env.ctx, env.logger, env.l2,
				common.HexToAddress(portalHex),
				crypto.PubkeyToAddress(l1Key.PublicKey),
				karsttest.NewBasePlan(l1Cl, l1Key),
				eth.OneHundredthEther,
			); err != nil {
				return fmt.Errorf("command error: %w", err)
			}
			return env.verifyFlashblocks()
		},
	}
}

// makeSpamCommand wires up the calldata spammer. Unlike the checks, it does not
// assert anything: it floods L2 with calldata-heavy transactions (the same shape
// the EIP-7934 block-size acceptance test uses) until interrupted with Ctrl+C,
// which is handy for manually driving up the L2 base fee or producing oversized
// blocks against an external network. Flashblocks tracking is not wired in
// because the spammer never includes its own "check" txs to verify.
func makeSpamCommand() *cli.Command {
	flags := []cli.Flag{EndpointL2, AccountKey, SpamAccounts, SpamRPS, SpamFundGwei, SpamBlockTime}
	flags = append(flags, oplog.CLIFlags(prefix)...)
	return &cli.Command{
		Name:  "spam",
		Usage: "Flood L2 with calldata-heavy transactions until interrupted (Ctrl+C).",
		Flags: cliapp.ProtectFlags(flags),
		Action: func(c *cli.Context) error {
			logCfg := oplog.ReadCLIConfig(c)
			logger := oplog.NewLogger(c.App.Writer, logCfg)

			c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
			l2Cl, err := ethclient.DialContext(c.Context, c.String(EndpointL2.Name))
			if err != nil {
				return fmt.Errorf("dial L2 RPC: %w", err)
			}
			defer l2Cl.Close()

			key, err := crypto.HexToECDSA(strings.TrimPrefix(c.String(AccountKey.Name), "0x"))
			if err != nil {
				return fmt.Errorf("parse account private key: %w", err)
			}

			return karsttest.SpamCalldata(c.Context, logger, l2Cl, key, karsttest.SpamConfig{
				NumAccounts:    c.Uint64(SpamAccounts.Name),
				FundPerAccount: eth.GWei(c.Uint64(SpamFundGwei.Name)),
				BaseRPS:        c.Uint64(SpamRPS.Name),
				To:             predeploys.L1BlockAddr,
				BlockTime:      c.Duration(SpamBlockTime.Name),
			})
		},
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "check-karst"
	app.Usage = "Check Karst upgrade results against an external network."
	app.Description = "Run post-Karst conformance checks against an external L2 RPC endpoint."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		makeAllCommand(),
		makeCommand("eip-7823", func(ctx context.Context, logger log.Logger, _ apis.EthCode, basePlan txplan.Option) error {
			_, _, err := karsttest.CheckEIP7823(ctx, logger, basePlan)
			return err
		}),
		makeCommand("eip-7883", func(ctx context.Context, logger log.Logger, _ apis.EthCode, basePlan txplan.Option) error {
			_, _, err := karsttest.CheckEIP7883(ctx, logger, basePlan)
			return err
		}),
		makeCommand("eip-7951", func(ctx context.Context, logger log.Logger, _ apis.EthCode, basePlan txplan.Option) error {
			_, _, err := karsttest.CheckEIP7951(ctx, logger, basePlan)
			return err
		}),
		makeCommand("karst-bn256-pair", func(ctx context.Context, logger log.Logger, _ apis.EthCode, basePlan txplan.Option) error {
			_, _, err := karsttest.CheckKarstBn256PairInputLimit(ctx, logger, basePlan)
			return err
		}),
		makeCommand("eip-7939", func(ctx context.Context, logger log.Logger, l2 apis.EthCode, basePlan txplan.Option) error {
			_, err := karsttest.CheckEIP7939(ctx, logger, l2, basePlan)
			return err
		}),
		makeCommand("eip-7825", func(ctx context.Context, logger log.Logger, _ apis.EthCode, basePlan txplan.Option) error {
			return karsttest.CheckEIP7825(ctx, logger, basePlan)
		}),
		makeBlockSizeCommand(),
		makeDepositCommand(),
		makeSpamCommand(),
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}
