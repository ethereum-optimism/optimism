package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
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
		Aliases: []string{"sdm-l2"},
		Usage:   "SDM producer execution RPC (admin and debug namespaces)",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_L2"),
		Value:   "http://localhost:9545",
	}
	SDMVerifierL2 = &cli.StringFlag{
		Name:    "sdm.l2-verifier",
		Aliases: []string{"sdm-l2-verifier"},
		Usage:   "Optional independent verifier execution RPC",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_L2_VERIFIER"),
	}
	SDMRollupRPC = &cli.StringFlag{
		Name:    "sdm.rollup-rpc",
		Aliases: []string{"sdm-rollup-rpc"},
		Usage:   "Optional rollup RPC for Lagoon activation and safe-head checks",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ROLLUP_RPC"),
	}
	SDMAccountKey = &cli.StringFlag{
		Name:    "sdm.account",
		Aliases: []string{"sdm-account"},
		Usage:   "Funded SDM workload private key; top-level all derives one when omitted",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ACCOUNT"),
	}
	SDMContract = &cli.StringFlag{
		Name:    "sdm.contract",
		Aliases: []string{"sdm-contract"},
		Usage:   "Existing StateBloat contract (deployed when omitted)",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_CONTRACT"),
	}
	SDMBlock = &cli.Uint64Flag{
		Name:    "sdm.block",
		Aliases: []string{"sdm-block"},
		Usage:   "Existing SDM block number to validate",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_BLOCK"),
	}
	SDMOptIn = &cli.BoolFlag{
		Name:    "sdm.opt-in",
		Aliases: []string{"sdm-opt-in"},
		Usage:   "Enable the producer's SDM opt-in for the duration of the check if it is off",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_OPT_IN"),
	}
	SDMJSON = &cli.BoolFlag{
		Name:    "sdm.json",
		Aliases: []string{"sdm-json"},
		Usage:   "Print the SDM result as JSON",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_JSON"),
	}
	AllSDMFundAmount = &cli.StringFlag{
		Name:    "sdm.l2-fund-amount",
		Aliases: []string{"sdm-l2-fund-amount"},
		Usage:   "Minimum SDM account balance in wei on each L2",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_L2_FUND_AMOUNT"),
		Value:   "20000000000000000",
	}
	AllSDMRollupRPCA = &cli.StringFlag{
		Name:    "sdm.rollup-rpc-a",
		Aliases: []string{"sdm-rollup-rpc-a"},
		Usage:   "Chain A rollup RPC",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ROLLUP_RPC_A"),
	}
	AllSDMRollupRPCB = &cli.StringFlag{
		Name:    "sdm.rollup-rpc-b",
		Aliases: []string{"sdm-rollup-rpc-b"},
		Usage:   "Chain B rollup RPC",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_ROLLUP_RPC_B"),
	}
	AllSDMContractA = &cli.StringFlag{
		Name:    "sdm.contract-a",
		Aliases: []string{"sdm-contract-a"},
		Usage:   "Existing StateBloat contract on chain A",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_CONTRACT_A"),
	}
	AllSDMContractB = &cli.StringFlag{
		Name:    "sdm.contract-b",
		Aliases: []string{"sdm-contract-b"},
		Usage:   "Existing StateBloat contract on chain B",
		EnvVars: op_service.PrefixEnvVar(prefix, "SDM_CONTRACT_B"),
	}
	AllJSON = &cli.BoolFlag{
		Name:    "json",
		Usage:   "Print the combined result as JSON",
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
	}, logcli.CLIFlags(prefix)...)
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
func setup(c *cli.Context, logOut io.Writer) (log.Logger, context.Context) {
	logCfg := logcli.ReadCLIConfig(c)
	logger := logcli.NewLogger(logOut, logCfg)
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
	logger, ctx := setup(c, c.App.Writer)
	cfg, err := baseConfig(ctx, logger, c)
	if err != nil {
		return err
	}
	defer cfg.Close()
	return checks.CheckRoundTrip(ctx, cfg, c.Int(Iterations.Name))
}

func failsafeAction(c *cli.Context) error {
	logger, ctx := setup(c, c.App.Writer)
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

const (
	allTimeout = 25 * time.Minute
	sdmTimeout = 10 * time.Minute
)

type lagoonAllResult struct {
	InteropPassed     bool             `json:"interop_passed"`
	InteropError      string           `json:"interop_error,omitempty"`
	SDMAccount        common.Address   `json:"sdm_account"`
	SDMAccountDerived bool             `json:"sdm_account_derived"`
	SDMChainA         *sdmcheck.Result `json:"sdm_chain_a"`
	SDMChainAError    string           `json:"sdm_chain_a_error,omitempty"`
	SDMChainB         *sdmcheck.Result `json:"sdm_chain_b"`
	SDMChainBError    string           `json:"sdm_chain_b_error,omitempty"`
}

func lagoonAllFlags() []cli.Flag {
	flags := []cli.Flag{
		ConfigFile,
		altsrc.NewStringFlag(EndpointL2A),
		altsrc.NewStringFlag(EndpointL2B),
		altsrc.NewStringFlag(AccountKey),
		altsrc.NewStringFlag(SDMAccountKey),
		altsrc.NewStringFlag(AllSDMFundAmount),
		altsrc.NewStringFlag(AllSDMRollupRPCA),
		altsrc.NewStringFlag(AllSDMRollupRPCB),
		altsrc.NewStringFlag(AllSDMContractA),
		altsrc.NewStringFlag(AllSDMContractB),
		altsrc.NewBoolFlag(SDMOptIn),
		altsrc.NewBoolFlag(AllJSON),
	}
	return append(flags, logcli.CLIFlags(prefix)...)
}

func parseOptionalAddressFlag(c *cli.Context, flag *cli.StringFlag) (*common.Address, error) {
	value := c.String(flag.Name)
	if value == "" {
		return nil, nil
	}
	if !common.IsHexAddress(value) {
		return nil, fmt.Errorf("--%s must be a hex address, got %q", flag.Name, value)
	}
	address := common.HexToAddress(value)
	return &address, nil
}

func lagoonAllAction(c *cli.Context) error {
	logger, ctx := setup(c, c.App.ErrWriter)
	ctx, cancel := context.WithTimeout(ctx, allTimeout)
	defer cancel()

	interopKeyHex := strings.TrimPrefix(c.String(AccountKey.Name), "0x")
	if interopKeyHex == "" {
		return errors.New("--account is required")
	}
	interopKey, err := crypto.HexToECDSA(interopKeyHex)
	if err != nil {
		return fmt.Errorf("parse --account: %w", err)
	}
	sdmKeyHex := strings.TrimPrefix(c.String(SDMAccountKey.Name), "0x")
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
			return fmt.Errorf("parse --sdm-account: %w", err)
		}
	}
	interopAccount := crypto.PubkeyToAddress(interopKey.PublicKey)
	sdmAccount := crypto.PubkeyToAddress(sdmKey.PublicKey)
	if interopAccount == sdmAccount {
		return errors.New("--account and --sdm-account must differ")
	}
	fundAmount, ok := new(big.Int).SetString(c.String(AllSDMFundAmount.Name), 0)
	if !ok || fundAmount.Sign() <= 0 {
		return fmt.Errorf("invalid --sdm-l2-fund-amount %q", c.String(AllSDMFundAmount.Name))
	}

	rollupAURL := c.String(AllSDMRollupRPCA.Name)
	rollupBURL := c.String(AllSDMRollupRPCB.Name)
	if rollupAURL == "" || rollupBURL == "" {
		return errors.New("--sdm-rollup-rpc-a and --sdm-rollup-rpc-b are required")
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
	baseSDM := sdmcheck.Config{Log: logger, Key: sdmKey, OptIn: c.Bool(SDMOptIn.Name)}
	cfgA := baseSDM
	cfgA.RPCURL = c.String(EndpointL2A.Name)
	cfgA.Contract = contractA
	cfgA.Rollup = rollupA
	cfgB := baseSDM
	cfgB.RPCURL = c.String(EndpointL2B.Name)
	cfgB.Contract = contractB
	cfgB.Rollup = rollupB

	logger.Info("funding SDM account", "account", sdmAccount, "minimumBalance", fundAmount, "derived", sdmAccountDerived)
	fundGroup, fundCtx := errgroup.WithContext(ctx)
	fundGroup.Go(func() error {
		if err := sdmcheck.EnsureFundedAccount(fundCtx, cfgA.RPCURL, interopKey, sdmAccount, fundAmount); err != nil {
			return fmt.Errorf("fund SDM account on chain A: %w", err)
		}
		return nil
	})
	fundGroup.Go(func() error {
		if err := sdmcheck.EnsureFundedAccount(fundCtx, cfgB.RPCURL, interopKey, sdmAccount, fundAmount); err != nil {
			return fmt.Errorf("fund SDM account on chain B: %w", err)
		}
		return nil
	})
	if err := fundGroup.Wait(); err != nil {
		return err
	}

	result := lagoonAllResult{SDMAccount: sdmAccount, SDMAccountDerived: sdmAccountDerived}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := interopsmoke.RunAll(groupCtx, c.App.ErrWriter, cfgA.RPCURL, cfgB.RPCURL, interopKeyHex); err != nil {
			result.InteropError = err.Error()
			return fmt.Errorf("Interop smoke: %w", err)
		}
		result.InteropPassed = true
		return nil
	})
	group.Go(func() error {
		check, err := sdmcheck.CheckAll(groupCtx, cfgA)
		result.SDMChainA = check
		if err != nil {
			result.SDMChainAError = err.Error()
			return fmt.Errorf("chain A SDM: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		check, err := sdmcheck.CheckAll(groupCtx, cfgB)
		result.SDMChainB = check
		if err != nil {
			result.SDMChainBError = err.Error()
			return fmt.Errorf("chain B SDM: %w", err)
		}
		return nil
	})
	groupErr := group.Wait()

	if c.Bool(AllJSON.Name) {
		encoder := json.NewEncoder(c.App.Writer)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return errors.Join(groupErr, err)
		}
	}
	if groupErr != nil {
		return groupErr
	}
	logger.Info("Lagoon smoke test passed",
		"sdmBlockA", result.SDMChainA.Conformance.Block.Number,
		"sdmBlockB", result.SDMChainB.Conformance.Block.Number)
	return nil
}

func sdmCheckFlags() []cli.Flag {
	return []cli.Flag{
		ConfigFile,
		altsrc.NewStringFlag(SDMEndpointL2),
		altsrc.NewStringFlag(SDMVerifierL2),
		altsrc.NewStringFlag(SDMRollupRPC),
		altsrc.NewBoolFlag(SDMJSON),
	}
}

func sdmAllFlags() []cli.Flag {
	flags := append(sdmCheckFlags(),
		altsrc.NewStringFlag(SDMAccountKey),
		altsrc.NewStringFlag(SDMContract),
		altsrc.NewBoolFlag(SDMOptIn),
	)
	return append(flags, logcli.CLIFlags(prefix)...)
}

func sdmBlockFlags() []cli.Flag {
	return append(append(sdmCheckFlags(), altsrc.NewUint64Flag(SDMBlock)), logcli.CLIFlags(prefix)...)
}

// resolveSDMConfig builds the config for a focused sdm subcommand; workload
// selects the sdm all flag set.
func resolveSDMConfig(ctx context.Context, logger log.Logger, c *cli.Context, workload bool) (sdmcheck.Config, func(), error) {
	cleanup := func() {}
	cfg := sdmcheck.Config{Log: logger, RPCURL: c.String(SDMEndpointL2.Name)}
	if workload {
		keyHex := strings.TrimPrefix(c.String(SDMAccountKey.Name), "0x")
		if keyHex == "" {
			return cfg, cleanup, errors.New("--sdm-account is required")
		}
		key, err := crypto.HexToECDSA(keyHex)
		if err != nil {
			return cfg, cleanup, fmt.Errorf("parse --sdm-account: %w", err)
		}
		cfg.Key = key
		cfg.OptIn = c.Bool(SDMOptIn.Name)
		contract, err := parseOptionalAddressFlag(c, SDMContract)
		if err != nil {
			return cfg, cleanup, err
		}
		cfg.Contract = contract
	}

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
	logger, ctx := setup(c, c.App.ErrWriter)
	ctx, cancel := context.WithTimeout(ctx, sdmTimeout)
	defer cancel()
	cfg, cleanup, err := resolveSDMConfig(ctx, logger, c, !existingBlock)
	if err != nil {
		return err
	}
	defer cleanup()
	if requireVerifier && cfg.Verifier == nil {
		return errors.New("--sdm-l2-verifier is required")
	}

	var result *sdmcheck.Result
	if existingBlock {
		blockNum := c.Uint64(SDMBlock.Name)
		if blockNum == 0 {
			return errors.New("--sdm-block is required")
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
		"refundEntries", len(validation.Payload.GasRefundEntries),
		"verifierChecked", result.VerifierChecked,
		"safeHeadChecked", result.SafeHeadChecked)
	return nil
}

func makeSDMCommand(tomlSource func(*cli.Context) (altsrc.InputSourceContext, error)) *cli.Command {
	allFlags := cliapp.ProtectFlags(sdmAllFlags())
	blockFlags := cliapp.ProtectFlags(sdmBlockFlags())
	verifierFlags := cliapp.ProtectFlags(sdmBlockFlags())
	return &cli.Command{
		Name:  "sdm",
		Usage: "Generate and validate SDM PostExec blocks",
		Subcommands: []*cli.Command{
			{
				Name:   "all",
				Usage:  "Generate load and validate the resulting PostExec block",
				Flags:  allFlags,
				Before: altsrc.InitInputSourceWithContext(allFlags, tomlSource),
				Action: func(c *cli.Context) error { return runSDMAction(c, false, false) },
			},
			{
				Name:   "block",
				Usage:  "Validate an existing SDM block",
				Flags:  blockFlags,
				Before: altsrc.InitInputSourceWithContext(blockFlags, tomlSource),
				Action: func(c *cli.Context) error { return runSDMAction(c, true, false) },
			},
			{
				Name:   "verifier",
				Usage:  "Validate an existing SDM block and verifier agreement",
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
	app.Usage = "Run Interop and SDM checks for the Lagoon upgrade."
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
			Usage:  "Run the Interop smoke suite and an SDM workload on both chains concurrently.",
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
