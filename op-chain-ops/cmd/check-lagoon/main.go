package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-lagoon/checks"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-lagoon/sdmcheck"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopsmoke"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

const prefix = "CHECK_LAGOON"

var (
	ConfigFile = &cli.StringFlag{
		Name:    "config",
		Usage:   "Path to a TOML config file supplying flag values. CLI flags and env vars override it.",
		EnvVars: op_service.PrefixEnvVar(prefix, "CONFIG"),
	}
	EndpointL2A = &cli.StringFlag{
		Name:    "l2-a",
		Usage:   "L2 chain A execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2_A"),
		Value:   "http://localhost:9545",
	}
	EndpointL2B = &cli.StringFlag{
		Name:    "l2-b",
		Usage:   "L2 chain B execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2_B"),
		Value:   "http://localhost:9546",
	}
	AccountKey = &cli.StringFlag{
		Name:    "account",
		Usage:   "Private key (hex-formatted string) of a funded test account on both chains",
		EnvVars: op_service.PrefixEnvVar(prefix, "ACCOUNT"),
	}
	FilterAdminRPC = &cli.StringSliceFlag{
		Name:    "filter.admin-rpc",
		Usage:   "op-interop-filter admin RPC URLs (repeat for multiple filters)",
		EnvVars: op_service.PrefixEnvVar(prefix, "FILTER_ADMIN_RPC"),
	}
	FilterJWTSecret = &cli.StringSliceFlag{
		Name:    "filter.jwt-secret",
		Usage:   "32-byte hex JWT secrets, one per --filter.admin-rpc entry (0x optional)",
		EnvVars: op_service.PrefixEnvVar(prefix, "FILTER_JWT_SECRET"),
	}
	RelayTimeout = &cli.DurationFlag{
		Name:    "relay-timeout",
		Usage:   "Maximum time to wait for a relayed cross-chain message (and other txs) to be included",
		EnvVars: op_service.PrefixEnvVar(prefix, "RELAY_TIMEOUT"),
		Value:   2 * time.Minute,
	}
	PropagationWait = &cli.DurationFlag{
		Name:    "propagation-wait",
		Usage:   "How long to wait for an initiating message to propagate before the failsafe-blocked attempt",
		EnvVars: op_service.PrefixEnvVar(prefix, "PROPAGATION_WAIT"),
		Value:   6 * time.Second,
	}
	Iterations = &cli.IntFlag{
		Name:    "iterations",
		Usage:   "Number of A->B->A round-trips to perform",
		EnvVars: op_service.PrefixEnvVar(prefix, "ITERATIONS"),
		Value:   3,
	}
	SDMEndpointL2 = &cli.StringFlag{
		Name:    "sdm.l2",
		Aliases: []string{"l2"},
		Usage:   "SDM producer execution RPC endpoint (admin and debug namespaces required)",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_L2"),
		Value:   "http://localhost:9545",
	}
	SDMVerifierL2 = &cli.StringFlag{
		Name:    "sdm.l2-verifier",
		Aliases: []string{"l2-verifier"},
		Usage:   "Optional independent verifier execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_L2_VERIFIER"),
	}
	SDMRollupRPC = &cli.StringFlag{
		Name:    "sdm.rollup-rpc",
		Aliases: []string{"rollup-rpc"},
		Usage:   "Optional rollup RPC used for Lagoon activation and safe-head checks",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ROLLUP_RPC"),
	}
	SDMAccountKey = &cli.StringFlag{
		Name:    "sdm.account",
		Aliases: []string{"account"},
		Usage:   "Private key of a funded SDM workload account",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ACCOUNT"),
	}
	SDMContract = &cli.StringFlag{
		Name:    "sdm.contract",
		Aliases: []string{"contract"},
		Usage:   "Existing StateBloat contract address (deploys one when omitted)",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_CONTRACT"),
	}
	SDMFundL2 = &cli.BoolFlag{
		Name:    "sdm.fund-l2",
		Aliases: []string{"fund-l2"},
		Usage:   "Fund an empty workload account through the OptimismPortal",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_FUND_L2"),
	}
	SDMEndpointL1 = &cli.StringFlag{
		Name:    "sdm.l1",
		Aliases: []string{"l1"},
		Usage:   "L1 execution RPC used with --fund-l2",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_L1"),
	}
	SDML1AccountKey = &cli.StringFlag{
		Name:    "sdm.l1-account",
		Aliases: []string{"l1-account"},
		Usage:   "Funded L1 account private key used with --fund-l2",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_L1_ACCOUNT"),
	}
	SDMPortal = &cli.StringFlag{
		Name:    "sdm.portal",
		Aliases: []string{"portal"},
		Usage:   "L1 OptimismPortalProxy address used with --fund-l2",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_PORTAL"),
	}
	SDMFundAmount = &cli.StringFlag{
		Name:    "sdm.fund-amount",
		Aliases: []string{"fund-amount"},
		Usage:   "Wei deposited when --fund-l2 is needed",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_FUND_AMOUNT"),
		Value:   "1000000000000000000",
	}
	SDMFundGasLimit = &cli.Uint64Flag{
		Name:    "sdm.fund-gas-limit",
		Aliases: []string{"fund-gas-limit"},
		Usage:   "L1 gas limit for the OptimismPortal deposit",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_FUND_GAS_LIMIT"),
		Value:   200_000,
	}
	SDMBlock = &cli.Uint64Flag{
		Name:    "sdm.block",
		Aliases: []string{"block"},
		Usage:   "Existing SDM block number to validate",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_BLOCK"),
	}
	SDMOptIn = &cli.BoolFlag{
		Name:    "sdm.opt-in",
		Aliases: []string{"opt-in"},
		Usage:   "Call admin_setOperatorSdmOptIn(true) before running the workload",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_OPT_IN"),
		Value:   true,
	}
	SDMBatchSize = &cli.IntFlag{
		Name:    "sdm.batch-size",
		Aliases: []string{"batch-size"},
		Usage:   "Number of StateBloat transactions submitted per attempt",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_BATCH_SIZE"),
		Value:   12,
	}
	SDMSlotCount = &cli.Uint64Flag{
		Name:    "sdm.slot-count",
		Aliases: []string{"slot-count"},
		Usage:   "Stable storage slots touched by each workload transaction",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_SLOT_COUNT"),
		Value:   20,
	}
	SDMMinUserTxs = &cli.IntFlag{
		Name:    "sdm.min-user-txs",
		Aliases: []string{"min-user-txs"},
		Usage:   "Minimum workload transactions required in the validated block",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_MIN_USER_TXS"),
		Value:   2,
	}
	SDMAttempts = &cli.IntFlag{
		Name:    "sdm.attempts",
		Aliases: []string{"attempts"},
		Usage:   "Maximum workload attempts",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ATTEMPTS"),
		Value:   3,
	}
	SDMGasLimit = &cli.Uint64Flag{
		Name:    "sdm.gas-limit",
		Aliases: []string{"gas-limit"},
		Usage:   "Gas limit for each workload transaction",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_GAS_LIMIT"),
		Value:   1_000_000,
	}
	SDMDeployGasLimit = &cli.Uint64Flag{
		Name:    "sdm.deploy-gas-limit",
		Aliases: []string{"deploy-gas-limit"},
		Usage:   "Gas limit for StateBloat deployment",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_DEPLOY_GAS_LIMIT"),
		Value:   2_000_000,
	}
	SDMTxSpacing = &cli.DurationFlag{
		Name:    "sdm.tx-spacing",
		Aliases: []string{"tx-spacing"},
		Usage:   "Delay between workload transaction submissions",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_TX_SPACING"),
	}
	SDMReceiptTimeout = &cli.DurationFlag{
		Name:    "sdm.receipt-timeout",
		Aliases: []string{"receipt-timeout"},
		Usage:   "Timeout for each workload receipt",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_RECEIPT_TIMEOUT"),
		Value:   45 * time.Second,
	}
	SDMVerifyTimeout = &cli.DurationFlag{
		Name:    "sdm.verify-timeout",
		Aliases: []string{"verify-timeout"},
		Usage:   "Timeout for verifier and safe-head agreement",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_VERIFY_TIMEOUT"),
		Value:   2 * time.Minute,
	}
	SDMTimeout = &cli.DurationFlag{
		Name:    "sdm.timeout",
		Aliases: []string{"timeout"},
		Usage:   "Overall SDM command timeout",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_TIMEOUT"),
		Value:   3 * time.Minute,
	}
	SDMJSON = &cli.BoolFlag{
		Name:    "sdm.json",
		Aliases: []string{"json"},
		Usage:   "Print the SDM result as JSON",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_JSON"),
	}
	AllSDMAccountKey = &cli.StringFlag{
		Name:    "sdm.account",
		Aliases: []string{"sdm-account"},
		Usage:   "Optional SDM private key; derives and funds a stable test account when omitted",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ACCOUNT"),
	}
	AllSDMFundAmount = &cli.StringFlag{
		Name:    "sdm.l2-fund-amount",
		Aliases: []string{"sdm-fund-amount"},
		Usage:   "Minimum wei balance funded to the SDM account on each L2",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_L2_FUND_AMOUNT"),
		Value:   "20000000000000000",
	}
	AllSDMRollupRPCA = &cli.StringFlag{
		Name:    "sdm.rollup-rpc-a",
		Aliases: []string{"sdm-rollup-rpc-a"},
		Usage:   "Chain A rollup RPC used for Lagoon activation and safe-head checks",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ROLLUP_RPC_A"),
	}
	AllSDMRollupRPCB = &cli.StringFlag{
		Name:    "sdm.rollup-rpc-b",
		Aliases: []string{"sdm-rollup-rpc-b"},
		Usage:   "Chain B rollup RPC used for Lagoon activation and safe-head checks",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ROLLUP_RPC_B"),
	}
	AllSDMContractA = &cli.StringFlag{
		Name:    "sdm.contract-a",
		Aliases: []string{"sdm-contract-a"},
		Usage:   "Existing StateBloat contract on chain A (deploys one when omitted)",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_CONTRACT_A"),
	}
	AllSDMContractB = &cli.StringFlag{
		Name:    "sdm.contract-b",
		Aliases: []string{"sdm-contract-b"},
		Usage:   "Existing StateBloat contract on chain B (deploys one when omitted)",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_CONTRACT_B"),
	}
	AllTimeout = &cli.DurationFlag{
		Name:    "all-timeout",
		Usage:   "Overall timeout for the concurrent Interop and SDM smoke test",
		EnvVars: op_service.PrefixEnvVar(prefix, "ALL_TIMEOUT"),
		Value:   25 * time.Minute,
	}
	AllJSON = &cli.BoolFlag{
		Name:    "json",
		Usage:   "Print the combined Lagoon result as JSON after the workloads complete",
		EnvVars: op_service.PrefixEnvVar(prefix, "ALL_JSON"),
	}
)

func baseFlags() []cli.Flag {
	return append([]cli.Flag{
		ConfigFile,
		altsrc.NewStringFlag(EndpointL2A),
		altsrc.NewStringFlag(EndpointL2B),
		altsrc.NewStringFlag(AccountKey),
		altsrc.NewDurationFlag(RelayTimeout),
	}, oplog.CLIFlags(prefix)...)
}

func roundtripFlags() []cli.Flag {
	return append(baseFlags(), altsrc.NewIntFlag(Iterations))
}

func failsafeFlags() []cli.Flag {
	return append(baseFlags(),
		altsrc.NewStringSliceFlag(FilterAdminRPC),
		altsrc.NewStringSliceFlag(FilterJWTSecret),
		altsrc.NewDurationFlag(PropagationWait),
	)
}

// setup reads the logging config and wires interrupt cancellation onto the context.
func setup(c *cli.Context) (log.Logger, context.Context) {
	logCfg := oplog.ReadCLIConfig(c)
	logger := oplog.NewLogger(c.App.Writer, logCfg)
	c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
	return logger, c.Context
}

// baseConfig dials both L2 clients, parses the test account, and verifies the
// two endpoints are distinct chains.
func baseConfig(ctx context.Context, logger log.Logger, c *cli.Context) (cfg *checks.CheckInteropConfig, err error) {
	var l2A, l2B *sources.EthClient
	defer func() {
		if err != nil {
			if l2A != nil {
				l2A.Close()
			}
			if l2B != nil {
				l2B.Close()
			}
		}
	}()

	l2A, err = dialEthClient(ctx, logger, c.String(EndpointL2A.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to dial L2 chain A: %w", err)
	}
	l2B, err = dialEthClient(ctx, logger, c.String(EndpointL2B.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to dial L2 chain B: %w", err)
	}
	keyHex := c.String(AccountKey.Name)
	if keyHex == "" {
		return nil, errors.New("test account private key is required: set --account, the env var, or 'account' in --config")
	}
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test private key: %w", err)
	}

	chainA, err := l2A.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 chain A id: %w", err)
	}
	chainB, err := l2B.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 chain B id: %w", err)
	}
	if chainA.Cmp(chainB) == 0 {
		return nil, fmt.Errorf("--l2-a and --l2-b must be different chains, both report chain id %s", chainA)
	}

	addr := crypto.PubkeyToAddress(key.PublicKey)
	logger.Info("running interop check", "chainA", chainA, "chainB", chainB, "account", addr)
	return &checks.CheckInteropConfig{
		Log:          logger,
		L2A:          l2A,
		L2B:          l2B,
		Key:          key,
		L2AChainID:   eth.ChainIDFromBig(chainA),
		L2BChainID:   eth.ChainIDFromBig(chainB),
		RelayTimeout: c.Duration(RelayTimeout.Name),
	}, nil
}

func dialEthClient(ctx context.Context, logger log.Logger, url string) (*sources.EthClient, error) {
	rpcCl, err := client.NewRPC(ctx, logger, url)
	if err != nil {
		return nil, err
	}
	return sources.NewEthClient(rpcCl, logger, nil, sources.DefaultEthClientConfig(10))
}

func dialFilterAdmins(ctx context.Context, logger log.Logger, urls, jwtSecrets []string) ([]client.RPC, error) {
	if len(urls) == 0 {
		return nil, errors.New("--filter.admin-rpc and --filter.jwt-secret are required for the failsafe check")
	}
	if len(urls) != len(jwtSecrets) {
		return nil, fmt.Errorf("--filter.admin-rpc and --filter.jwt-secret must have equal counts (got %d and %d)", len(urls), len(jwtSecrets))
	}
	clients := make([]client.RPC, 0, len(urls))
	for i, url := range urls {
		raw, err := hex.DecodeString(strings.TrimPrefix(jwtSecrets[i], "0x"))
		if err != nil || len(raw) != 32 {
			for _, c := range clients {
				c.Close()
			}
			return nil, fmt.Errorf("filter jwt secret %d: must be a 32-byte hex string", i)
		}
		cl, err := client.NewRPC(ctx, logger, url,
			client.WithGethRPCOptions(gethrpc.WithHTTPAuth(gn.NewJWTAuth([32]byte(raw)))))
		if err != nil {
			for _, c := range clients {
				c.Close()
			}
			return nil, fmt.Errorf("dial filter admin %d (%s): %w", i, url, err)
		}
		clients = append(clients, cl)
	}
	return clients, nil
}

func roundTripAction(c *cli.Context) error {
	logger, ctx := setup(c)
	cfg, err := baseConfig(ctx, logger, c)
	if err != nil {
		return err
	}
	defer cfg.Close()
	return checks.CheckRoundTrip(ctx, cfg, c.Int(Iterations.Name))
}

func failsafeAction(c *cli.Context) error {
	logger, ctx := setup(c)
	cfg, err := baseConfig(ctx, logger, c)
	if err != nil {
		return err
	}
	defer cfg.Close()
	admins, err := dialFilterAdmins(ctx, logger, c.StringSlice(FilterAdminRPC.Name), c.StringSlice(FilterJWTSecret.Name))
	if err != nil {
		return err
	}
	cfg.FilterAdmins = admins
	cfg.PropagationWait = c.Duration(PropagationWait.Name)
	return checks.CheckFailsafe(ctx, cfg)
}

// lagoonAllResult is emitted by the top-level all command after all three
// concurrent workloads complete.
type lagoonAllResult struct {
	InteropPassed     bool             `json:"interop_passed"`
	SDMAccount        common.Address   `json:"sdm_account"`
	SDMAccountDerived bool             `json:"sdm_account_derived"`
	SDMChainA         *sdmcheck.Result `json:"sdm_chain_a"`
	SDMChainB         *sdmcheck.Result `json:"sdm_chain_b"`
}

func lagoonAllFlags() []cli.Flag {
	flags := []cli.Flag{
		ConfigFile,
		altsrc.NewStringFlag(EndpointL2A),
		altsrc.NewStringFlag(EndpointL2B),
		altsrc.NewStringFlag(AccountKey),
		altsrc.NewStringFlag(AllSDMAccountKey),
		altsrc.NewStringFlag(AllSDMFundAmount),
		altsrc.NewStringFlag(AllSDMRollupRPCA),
		altsrc.NewStringFlag(AllSDMRollupRPCB),
		altsrc.NewStringFlag(AllSDMContractA),
		altsrc.NewStringFlag(AllSDMContractB),
		altsrc.NewBoolFlag(SDMOptIn),
		altsrc.NewIntFlag(SDMBatchSize),
		altsrc.NewUint64Flag(SDMSlotCount),
		altsrc.NewIntFlag(SDMMinUserTxs),
		altsrc.NewIntFlag(SDMAttempts),
		altsrc.NewUint64Flag(SDMGasLimit),
		altsrc.NewUint64Flag(SDMDeployGasLimit),
		altsrc.NewDurationFlag(SDMTxSpacing),
		altsrc.NewDurationFlag(SDMReceiptTimeout),
		altsrc.NewDurationFlag(SDMVerifyTimeout),
		altsrc.NewDurationFlag(AllTimeout),
		altsrc.NewBoolFlag(AllJSON),
	}
	return append(flags, oplog.CLIFlags(prefix)...)
}

func parseOptionalAddressFlag(c *cli.Context, flag *cli.StringFlag) (*common.Address, error) {
	value := c.String(flag.Name)
	if value == "" {
		return nil, nil
	}
	if !common.IsHexAddress(value) {
		return nil, fmt.Errorf("--%s must be a hex address, got %q", flag.Aliases[0], value)
	}
	address := common.HexToAddress(value)
	return &address, nil
}

func lagoonAllAction(c *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(c)
	logger := oplog.NewLogger(c.App.ErrWriter, logCfg)
	c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
	ctx, cancel := context.WithTimeout(c.Context, c.Duration(AllTimeout.Name))
	defer cancel()

	interopKeyHex := strings.TrimPrefix(c.String(AccountKey.Name), "0x")
	if interopKeyHex == "" {
		return errors.New("combined Lagoon smoke requires --account for Interop transactions")
	}
	interopKey, err := crypto.HexToECDSA(interopKeyHex)
	if err != nil {
		return fmt.Errorf("parse Interop account private key: %w", err)
	}
	sdmKeyHex := strings.TrimPrefix(c.String(AllSDMAccountKey.Name), "0x")
	sdmAccountDerived := sdmKeyHex == ""
	var sdmKey *ecdsa.PrivateKey
	if sdmAccountDerived {
		sdmKey, err = sdmcheck.DeriveSDMAccountKey(interopKey)
		if err != nil {
			return fmt.Errorf("derive SDM account: %w", err)
		}
	} else {
		sdmKey, err = crypto.HexToECDSA(sdmKeyHex)
		if err != nil {
			return fmt.Errorf("parse SDM account private key: %w", err)
		}
	}
	interopAccount := crypto.PubkeyToAddress(interopKey.PublicKey)
	sdmAccount := crypto.PubkeyToAddress(sdmKey.PublicKey)
	if interopAccount == sdmAccount {
		return errors.New("--account and --sdm-account must differ to avoid concurrent nonce contention")
	}
	fundAmount, ok := new(big.Int).SetString(c.String(AllSDMFundAmount.Name), 0)
	if !ok || fundAmount.Sign() <= 0 {
		return fmt.Errorf("invalid --sdm-fund-amount %q", c.String(AllSDMFundAmount.Name))
	}

	rollupAURL := c.String(AllSDMRollupRPCA.Name)
	rollupBURL := c.String(AllSDMRollupRPCB.Name)
	if rollupAURL == "" || rollupBURL == "" {
		return errors.New("combined Lagoon smoke requires --sdm-rollup-rpc-a and --sdm-rollup-rpc-b")
	}
	rollupARPC, err := client.NewRPC(ctx, logger, rollupAURL)
	if err != nil {
		return fmt.Errorf("dial chain A rollup RPC: %w", err)
	}
	rollupA := sources.NewRollupClient(rollupARPC)
	defer rollupA.Close()
	rollupBRPC, err := client.NewRPC(ctx, logger, rollupBURL)
	if err != nil {
		return fmt.Errorf("dial chain B rollup RPC: %w", err)
	}
	rollupB := sources.NewRollupClient(rollupBRPC)
	defer rollupB.Close()

	contractA, err := parseOptionalAddressFlag(c, AllSDMContractA)
	if err != nil {
		return err
	}
	contractB, err := parseOptionalAddressFlag(c, AllSDMContractB)
	if err != nil {
		return err
	}
	baseSDM := sdmcheck.Config{
		Log:            logger,
		Key:            sdmKey,
		OptIn:          c.Bool(SDMOptIn.Name),
		BatchSize:      c.Int(SDMBatchSize.Name),
		SlotCount:      c.Uint64(SDMSlotCount.Name),
		MinUserTxs:     c.Int(SDMMinUserTxs.Name),
		Attempts:       c.Int(SDMAttempts.Name),
		GasLimit:       c.Uint64(SDMGasLimit.Name),
		DeployLimit:    c.Uint64(SDMDeployGasLimit.Name),
		TxSpacing:      c.Duration(SDMTxSpacing.Name),
		ReceiptTimeout: c.Duration(SDMReceiptTimeout.Name),
		VerifyTimeout:  c.Duration(SDMVerifyTimeout.Name),
	}
	cfgA := baseSDM
	cfgA.RPCURL = c.String(EndpointL2A.Name)
	cfgA.Contract = contractA
	cfgA.Rollup = rollupA
	cfgB := baseSDM
	cfgB.RPCURL = c.String(EndpointL2B.Name)
	cfgB.Contract = contractB
	cfgB.Rollup = rollupB

	logger.Info("funding SDM account on both L2s",
		"account", sdmAccount, "minimumBalance", fundAmount, "derived", sdmAccountDerived)
	fundGroup, fundCtx := errgroup.WithContext(ctx)
	fundGroup.Go(func() error {
		if err := sdmcheck.EnsureFundedAccount(fundCtx, cfgA.RPCURL, interopKey, sdmAccount, fundAmount, cfgA.ReceiptTimeout, cfgA.PollInterval); err != nil {
			return fmt.Errorf("fund SDM account on chain A: %w", err)
		}
		return nil
	})
	fundGroup.Go(func() error {
		if err := sdmcheck.EnsureFundedAccount(fundCtx, cfgB.RPCURL, interopKey, sdmAccount, fundAmount, cfgB.ReceiptTimeout, cfgB.PollInterval); err != nil {
			return fmt.Errorf("fund SDM account on chain B: %w", err)
		}
		return nil
	})
	if err := fundGroup.Wait(); err != nil {
		return err
	}

	logger.Info("starting concurrent Lagoon smoke test",
		"l2A", cfgA.RPCURL, "l2B", cfgB.RPCURL,
		"interopAccount", interopAccount,
		"sdmAccount", sdmAccount)
	result := lagoonAllResult{SDMAccount: sdmAccount, SDMAccountDerived: sdmAccountDerived}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := interopsmoke.RunAll(groupCtx, c.App.ErrWriter, cfgA.RPCURL, cfgB.RPCURL, interopKeyHex); err != nil {
			return fmt.Errorf("Interop smoke: %w", err)
		}
		result.InteropPassed = true
		return nil
	})
	group.Go(func() error {
		check, err := sdmcheck.CheckAll(groupCtx, cfgA)
		result.SDMChainA = check
		if err != nil {
			return fmt.Errorf("chain A SDM smoke: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		check, err := sdmcheck.CheckAll(groupCtx, cfgB)
		result.SDMChainB = check
		if err != nil {
			return fmt.Errorf("chain B SDM smoke: %w", err)
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return err
	}

	if c.Bool(AllJSON.Name) {
		encoder := json.NewEncoder(c.App.Writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	logger.Info("combined Lagoon smoke test passed",
		"interop", result.InteropPassed,
		"sdmBlockA", result.SDMChainA.Conformance.Block.Number,
		"sdmBlockB", result.SDMChainB.Conformance.Block.Number)
	return nil
}

func sdmFlags(includeBlock bool) []cli.Flag {
	flags := []cli.Flag{
		ConfigFile,
		altsrc.NewStringFlag(SDMEndpointL2),
		altsrc.NewStringFlag(SDMVerifierL2),
		altsrc.NewStringFlag(SDMRollupRPC),
		altsrc.NewStringFlag(SDMAccountKey),
		altsrc.NewStringFlag(SDMContract),
		altsrc.NewBoolFlag(SDMFundL2),
		altsrc.NewStringFlag(SDMEndpointL1),
		altsrc.NewStringFlag(SDML1AccountKey),
		altsrc.NewStringFlag(SDMPortal),
		altsrc.NewStringFlag(SDMFundAmount),
		altsrc.NewUint64Flag(SDMFundGasLimit),
		altsrc.NewBoolFlag(SDMOptIn),
		altsrc.NewIntFlag(SDMBatchSize),
		altsrc.NewUint64Flag(SDMSlotCount),
		altsrc.NewIntFlag(SDMMinUserTxs),
		altsrc.NewIntFlag(SDMAttempts),
		altsrc.NewUint64Flag(SDMGasLimit),
		altsrc.NewUint64Flag(SDMDeployGasLimit),
		altsrc.NewDurationFlag(SDMTxSpacing),
		altsrc.NewDurationFlag(SDMReceiptTimeout),
		altsrc.NewDurationFlag(SDMVerifyTimeout),
		altsrc.NewDurationFlag(SDMTimeout),
		altsrc.NewBoolFlag(SDMJSON),
	}
	if includeBlock {
		flags = append(flags, altsrc.NewUint64Flag(SDMBlock))
	}
	return append(flags, oplog.CLIFlags(prefix)...)
}

func resolveSDMConfig(ctx context.Context, logger log.Logger, c *cli.Context, requireKey bool) (sdmcheck.Config, func(), error) {
	cleanup := func() {}
	cfg := sdmcheck.Config{
		Log:            logger,
		RPCURL:         c.String(SDMEndpointL2.Name),
		OptIn:          c.Bool(SDMOptIn.Name),
		BatchSize:      c.Int(SDMBatchSize.Name),
		SlotCount:      c.Uint64(SDMSlotCount.Name),
		MinUserTxs:     c.Int(SDMMinUserTxs.Name),
		Attempts:       c.Int(SDMAttempts.Name),
		GasLimit:       c.Uint64(SDMGasLimit.Name),
		DeployLimit:    c.Uint64(SDMDeployGasLimit.Name),
		TxSpacing:      c.Duration(SDMTxSpacing.Name),
		ReceiptTimeout: c.Duration(SDMReceiptTimeout.Name),
		VerifyTimeout:  c.Duration(SDMVerifyTimeout.Name),
		FundL2:         c.Bool(SDMFundL2.Name),
		L1RPCURL:       c.String(SDMEndpointL1.Name),
		FundGasLimit:   c.Uint64(SDMFundGasLimit.Name),
	}
	if keyHex := strings.TrimPrefix(c.String(SDMAccountKey.Name), "0x"); keyHex != "" {
		key, err := crypto.HexToECDSA(keyHex)
		if err != nil {
			return cfg, cleanup, fmt.Errorf("parse SDM account private key: %w", err)
		}
		cfg.Key = key
	} else if requireKey {
		return cfg, cleanup, errors.New("SDM workload requires --account, CHECK_LAGOON_SDM_ACCOUNT, or sdm.account in --config")
	}
	if contractHex := c.String(SDMContract.Name); contractHex != "" {
		if !common.IsHexAddress(contractHex) {
			return cfg, cleanup, fmt.Errorf("invalid SDM contract address %q", contractHex)
		}
		contract := common.HexToAddress(contractHex)
		cfg.Contract = &contract
	}
	if l1KeyHex := strings.TrimPrefix(c.String(SDML1AccountKey.Name), "0x"); l1KeyHex != "" {
		key, err := crypto.HexToECDSA(l1KeyHex)
		if err != nil {
			return cfg, cleanup, fmt.Errorf("parse SDM L1 account private key: %w", err)
		}
		cfg.L1Key = key
	}
	if portalHex := c.String(SDMPortal.Name); portalHex != "" {
		if !common.IsHexAddress(portalHex) {
			return cfg, cleanup, fmt.Errorf("invalid SDM OptimismPortal address %q", portalHex)
		}
		cfg.Portal = common.HexToAddress(portalHex)
	}
	fundAmount, ok := new(big.Int).SetString(c.String(SDMFundAmount.Name), 0)
	if !ok || fundAmount.Sign() <= 0 {
		return cfg, cleanup, fmt.Errorf("invalid SDM funding amount %q", c.String(SDMFundAmount.Name))
	}
	cfg.FundAmount = fundAmount

	closers := make([]func(), 0, 2)
	cleanup = func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i]()
		}
	}
	if verifierURL := c.String(SDMVerifierL2.Name); verifierURL != "" {
		verifierRPC, err := client.NewRPC(ctx, logger, verifierURL)
		if err != nil {
			cleanup()
			return cfg, func() {}, fmt.Errorf("dial SDM verifier RPC: %w", err)
		}
		closers = append(closers, verifierRPC.Close)
		cfg.Verifier = verifierRPC
	}
	if rollupURL := c.String(SDMRollupRPC.Name); rollupURL != "" {
		rollupRPC, err := client.NewRPC(ctx, logger, rollupURL)
		if err != nil {
			cleanup()
			return cfg, func() {}, fmt.Errorf("dial SDM rollup RPC: %w", err)
		}
		rollupClient := sources.NewRollupClient(rollupRPC)
		closers = append(closers, rollupClient.Close)
		cfg.Rollup = rollupClient
	}
	return cfg, cleanup, nil
}

func runSDMAction(c *cli.Context, existingBlock bool, requireVerifier bool) error {
	logCfg := oplog.ReadCLIConfig(c)
	logger := oplog.NewLogger(c.App.ErrWriter, logCfg)
	c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
	ctx := c.Context
	ctx, cancel := context.WithTimeout(ctx, c.Duration(SDMTimeout.Name))
	defer cancel()
	cfg, cleanup, err := resolveSDMConfig(ctx, logger, c, !existingBlock)
	if err != nil {
		return err
	}
	defer cleanup()
	if requireVerifier && cfg.Verifier == nil {
		return errors.New("sdm verifier requires --l2-verifier")
	}

	var result *sdmcheck.Result
	if existingBlock {
		blockNum := c.Uint64(SDMBlock.Name)
		if blockNum == 0 {
			return errors.New("sdm block check requires --block")
		}
		result, err = sdmcheck.CheckBlock(ctx, cfg, blockNum)
	} else {
		result, err = sdmcheck.CheckAll(ctx, cfg)
	}
	if err != nil {
		return err
	}
	if c.Bool(SDMJSON.Name) {
		encoder := json.NewEncoder(c.App.Writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	validation := result.Conformance.ValidationResult
	logger.Info("SDM conformance check passed",
		"block", validation.Block.Number,
		"hash", validation.Block.Hash,
		"transactions", len(validation.Block.Transactions),
		"refundEntries", len(validation.Payload.GasRefundEntries),
		"totalRefund", validation.TotalPayloadRefund,
		"verifierChecked", result.VerifierChecked,
		"safeHeadChecked", result.SafeHeadChecked)
	return nil
}

func makeSDMCommand(tomlSource func(*cli.Context) (altsrc.InputSourceContext, error)) *cli.Command {
	allFlags := cliapp.ProtectFlags(sdmFlags(false))
	blockFlags := cliapp.ProtectFlags(sdmFlags(true))
	verifierFlags := cliapp.ProtectFlags(sdmFlags(true))
	return &cli.Command{
		Name:  "sdm",
		Usage: "Generate and validate SDM PostExec transactions",
		Subcommands: []*cli.Command{
			{
				Name:   "all",
				Usage:  "Opt in, generate non-trivial load, and run every configured SDM check",
				Flags:  allFlags,
				Before: altsrc.InitInputSourceWithContext(allFlags, tomlSource),
				Action: func(c *cli.Context) error { return runSDMAction(c, false, false) },
			},
			{
				Name:   "block",
				Usage:  "Validate an existing SDM block without submitting transactions",
				Flags:  blockFlags,
				Before: altsrc.InitInputSourceWithContext(blockFlags, tomlSource),
				Action: func(c *cli.Context) error { return runSDMAction(c, true, false) },
			},
			{
				Name:   "verifier",
				Usage:  "Validate producer and independent verifier agreement for an SDM block",
				Flags:  verifierFlags,
				Before: altsrc.InitInputSourceWithContext(verifierFlags, tomlSource),
				Action: func(c *cli.Context) error { return runSDMAction(c, true, true) },
			},
		},
	}
}

func newApp() *cli.App {
	app := cli.NewApp()
	app.Name = "check-lagoon"
	app.Usage = "Run Interop and SDM conformance checks for the Lagoon upgrade."
	app.Description = "Smoke-test Interop messaging, the Interop filter failsafe, and SDM PostExec conformance."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	allCmdFlags := cliapp.ProtectFlags(lagoonAllFlags())
	roundtripCmdFlags := cliapp.ProtectFlags(roundtripFlags())
	failsafeCmdFlags := cliapp.ProtectFlags(failsafeFlags())
	tomlSource := altsrc.NewTomlSourceFromFlagFunc(ConfigFile.Name)
	app.Commands = []*cli.Command{
		{
			Name:   "all",
			Usage:  "Run full Interop smoke and SDM workloads on both chains concurrently.",
			Flags:  allCmdFlags,
			Before: altsrc.InitInputSourceWithContext(allCmdFlags, tomlSource),
			Action: lagoonAllAction,
		},
		{
			Name:   "roundtrip",
			Usage:  "Send an interop message A -> B and B -> A, relaying each.",
			Flags:  roundtripCmdFlags,
			Before: altsrc.InitInputSourceWithContext(roundtripCmdFlags, tomlSource),
			Action: roundTripAction,
		},
		{
			Name:   "failsafe",
			Usage:  "Verify interop messages succeed, are blocked while failsafe is enabled, then succeed again.",
			Flags:  failsafeCmdFlags,
			Before: altsrc.InitInputSourceWithContext(failsafeCmdFlags, tomlSource),
			Action: failsafeAction,
		},
		makeSDMCommand(tomlSource),
	}

	return app
}

func main() {
	if err := newApp().Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}
