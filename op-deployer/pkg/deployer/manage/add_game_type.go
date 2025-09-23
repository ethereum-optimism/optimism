package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"

	"github.com/ethereum-optimism/optimism/op-service/cliutil"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

type AddGameTypeConfig struct {
	L1RPCUrl                string
	Logger                  log.Logger
	ArtifactsLocator        *artifacts.Locator
	CacheDir                string
	L1ProxyAdminOwner       common.Address
	OPCMImpl                common.Address
	SystemConfigProxy       common.Address
	OPChainProxyAdmin       common.Address
	DelayedWETHProxy        common.Address
	DisputeGameType         uint32
	DisputeAbsolutePrestate common.Hash
	DisputeMaxGameDepth     *big.Int
	DisputeSplitDepth       *big.Int
	DisputeClockExtension   uint64
	DisputeMaxClockDuration uint64
	InitialBond             *big.Int
	VM                      common.Address
	Permissionless          bool
	SaltMixer               string
}

func (c *AddGameTypeConfig) Check() error {
	if c.L1RPCUrl == "" {
		return fmt.Errorf("l1RPCUrl must be specified")
	}

	if c.Logger == nil {
		return fmt.Errorf("logger must be specified")
	}

	if c.ArtifactsLocator == nil {
		return fmt.Errorf("artifactsLocator must be specified")
	}

	if c.CacheDir == "" {
		return fmt.Errorf("cacheDir must be specified")
	}

	if c.L1ProxyAdminOwner == (common.Address{}) {
		return fmt.Errorf("prank address must be specified")
	}

	if c.OPCMImpl == (common.Address{}) {
		return fmt.Errorf("opcmImpl address must be specified")
	}

	if c.SystemConfigProxy == (common.Address{}) {
		return fmt.Errorf("systemConfigProxy address must be specified")
	}

	if c.OPChainProxyAdmin == (common.Address{}) {
		return fmt.Errorf("opChainProxyAdmin address must be specified")
	}

	if c.DisputeAbsolutePrestate == (common.Hash{}) {
		return fmt.Errorf("disputeAbsolutePrestate must be specified")
	}

	if c.DisputeMaxGameDepth == nil || c.DisputeMaxGameDepth.Sign() == 0 {
		return fmt.Errorf("disputeMaxGameDepth must be non-zero")
	}

	if c.DisputeSplitDepth == nil || c.DisputeSplitDepth.Sign() == 0 {
		return fmt.Errorf("disputeSplitDepth must be non-zero")
	}

	if c.DisputeClockExtension == 0 {
		return fmt.Errorf("disputeClockExtension must be non-zero")
	}

	if c.DisputeMaxClockDuration == 0 {
		return fmt.Errorf("disputeMaxClockDuration must be non-zero")
	}

	if c.InitialBond == nil || c.InitialBond.Sign() == 0 {
		return fmt.Errorf("initialBond must be non-zero")
	}

	if c.SaltMixer == "" {
		return fmt.Errorf("saltMixer must be specified")
	}

	return nil
}

func AddGameTypeCLI(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	artifactsLocatorStr := cliCtx.String(deployer.ArtifactsLocatorFlag.Name)
	artifactsLocator := new(artifacts.Locator)
	if err := artifactsLocator.UnmarshalText([]byte(artifactsLocatorStr)); err != nil {
		return fmt.Errorf("failed to parse artifacts locator: %w", err)
	}

	cfg := AddGameTypeConfig{
		L1RPCUrl:                cliCtx.String(deployer.L1RPCURLFlagName),
		Logger:                  l,
		ArtifactsLocator:        artifactsLocator,
		CacheDir:                cliCtx.String(deployer.CacheDirFlag.Name),
		DisputeGameType:         uint32(cliCtx.Uint64(DisputeGameTypeFlag.Name)),
		DisputeAbsolutePrestate: common.HexToHash(cliCtx.String(DisputeAbsolutePrestateFlag.Name)),
		DisputeMaxGameDepth:     new(big.Int).SetUint64(cliCtx.Uint64(DisputeMaxGameDepthFlag.Name)),
		DisputeSplitDepth:       new(big.Int).SetUint64(cliCtx.Uint64(DisputeSplitDepthFlag.Name)),
		DisputeClockExtension:   cliCtx.Uint64(DisputeClockExtensionFlag.Name),
		DisputeMaxClockDuration: cliCtx.Uint64(DisputeMaxClockDurationFlag.Name),
		Permissionless:          cliCtx.Bool(PermissionlessFlag.Name),
		SaltMixer:               cliCtx.String(SaltMixerFlag.Name),
		DelayedWETHProxy:        common.HexToAddress(cliCtx.String(DelayedWETHProxyFlag.Name)),
	}

	var err error
	if cliCtx.IsSet(WorkdirFlag.Name) {
		err = populateConfigFromWorkdir(&cfg, cliCtx)
	} else {
		err = populateConfigFromFlags(&cfg, cliCtx)
	}
	if err != nil {
		return err
	}

	initialBond, err := cliutil.BigIntFlag(cliCtx, InitialBondFlag.Name)
	if err != nil {
		cfg.InitialBond = initialBond
	} else {
		return fmt.Errorf("failed to parse initial bond: %w", err)
	}

	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)

	_, calldata, err := AddGameType(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to add game type: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(calldata); err != nil {
		return fmt.Errorf("failed to encode calldata: %w", err)
	}

	return nil
}

func populateConfigFromWorkdir(cfg *AddGameTypeConfig, cliCtx *cli.Context) error {
	workdir := cliCtx.String(WorkdirFlag.Name)
	chainIDStr := cliCtx.String(L2ChainIDFlag.Name)

	if chainIDStr == "" {
		return fmt.Errorf("flag --%s must be specified when --workdir is set", L2ChainIDFlag.Name)
	}
	chainID, err := eth.ChainIDFromString(chainIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse chain-id: %w", err)
	}

	// Ensure these address flags are not provided if the workdir flag is provided
	addressFlags := []string{
		L1ProxyAdminOwnerFlag.Name, OPCMImplFlag.Name, SystemConfigProxyFlag.Name,
		OPChainProxyAdminFlag.Name, DelayedWETHProxyFlag.Name, VMFlag.Name,
	}
	for _, flagName := range addressFlags {
		if cliCtx.String(flagName) != "" {
			return fmt.Errorf("cannot specify --%s when --%s is set", flagName, WorkdirFlag.Name)
		}
	}

	state, err := pipeline.ReadState(workdir)
	if err != nil {
		return fmt.Errorf("failed to read state from %s: %w", workdir, err)
	}
	if state.AppliedIntent == nil {
		return fmt.Errorf("no applied intent in state file %s", workdir)
	}
	chainState, err := state.Chain(chainID.Bytes32())
	if err != nil {
		return fmt.Errorf("failed to get chain config for chain ID %s from state file %s: %w", chainIDStr, workdir, err)
	}
	chainIntent, err := state.AppliedIntent.Chain(chainID.Bytes32())
	if err != nil {
		return fmt.Errorf("failed to get applied chain intent for chain ID %s from state file %s: %w", chainIDStr, workdir, err)
	}

	cfg.L1ProxyAdminOwner = chainIntent.Roles.L1ProxyAdminOwner
	if state.AppliedIntent.OPCMAddress == nil {
		return fmt.Errorf("OPCMAddress is not set in the applied intent of state file %s", workdir)
	}
	cfg.OPCMImpl = *state.AppliedIntent.OPCMAddress
	cfg.SystemConfigProxy = chainState.SystemConfigProxy
	cfg.OPChainProxyAdmin = chainState.OpChainProxyAdminImpl
	cfg.VM = state.ImplementationsDeployment.MipsImpl
	return nil
}

func populateConfigFromFlags(cfg *AddGameTypeConfig, cliCtx *cli.Context) error {
	if cliCtx.String(L2ChainIDFlag.Name) != "" {
		return fmt.Errorf("--l2-chain-id must not be specified when workdir is not set")
	}

	cfg.L1ProxyAdminOwner = common.HexToAddress(cliCtx.String(L1ProxyAdminOwnerFlag.Name))
	cfg.OPCMImpl = common.HexToAddress(cliCtx.String(OPCMImplFlag.Name))
	cfg.SystemConfigProxy = common.HexToAddress(cliCtx.String(SystemConfigProxyFlag.Name))
	cfg.OPChainProxyAdmin = common.HexToAddress(cliCtx.String(OPChainProxyAdminFlag.Name))
	cfg.VM = common.HexToAddress(cliCtx.String(VMFlag.Name))
	return nil
}

func AddGameType(ctx context.Context, cfg AddGameTypeConfig) (opcm.AddGameTypeOutput, []broadcaster.CalldataDump, error) {
	var output opcm.AddGameTypeOutput
	if err := cfg.Check(); err != nil {
		return output, nil, fmt.Errorf("invalid config for AddGameType: %w", err)
	}

	lgr := cfg.Logger

	artifactsFS, err := artifacts.Download(ctx, cfg.ArtifactsLocator, ioutil.BarProgressor(), cfg.CacheDir)
	if err != nil {
		return output, nil, fmt.Errorf("failed to download artifacts: %w", err)
	}

	bcaster := new(broadcaster.CalldataBroadcaster)

	l1RPC, err := rpc.Dial(cfg.L1RPCUrl)
	if err != nil {
		return output, nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}

	l1Host, err := env.DefaultForkedScriptHost(
		ctx,
		bcaster,
		lgr,
		common.Address{'D'},
		artifactsFS,
		l1RPC,
	)
	if err != nil {
		return output, nil, fmt.Errorf("failed to create script host: %w", err)
	}

	script, err := opcm.NewAddGameTypeScript(l1Host)
	if err != nil {
		return output, nil, fmt.Errorf("failed to create L2 genesis script: %w", err)
	}

	output, err = script.Run(opcm.AddGameTypeInput{
		L1ProxyAdminOwner:       cfg.L1ProxyAdminOwner,
		OPCMImpl:                cfg.OPCMImpl,
		SystemConfigProxy:       cfg.SystemConfigProxy,
		OPChainProxyAdmin:       cfg.OPChainProxyAdmin,
		DelayedWETHProxy:        cfg.DelayedWETHProxy,
		DisputeGameType:         cfg.DisputeGameType,
		DisputeAbsolutePrestate: cfg.DisputeAbsolutePrestate,
		DisputeMaxGameDepth:     cfg.DisputeMaxGameDepth,
		DisputeSplitDepth:       cfg.DisputeSplitDepth,
		DisputeClockExtension:   cfg.DisputeClockExtension,
		DisputeMaxClockDuration: cfg.DisputeMaxClockDuration,
		InitialBond:             cfg.InitialBond,
		VM:                      cfg.VM,
		SaltMixer:               cfg.SaltMixer,
		Permissioned:            !cfg.Permissionless,
	})
	if err != nil {
		return output, nil, fmt.Errorf("error adding game type: %w", err)
	}

	// Get the calldata
	calldata, err := bcaster.Dump()
	if err != nil {
		return output, nil, fmt.Errorf("failed to get calldata: %w", err)
	}

	return output, calldata, nil
}
