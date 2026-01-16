package flags

import (
	"fmt"
	"net/url"
	"runtime"
	"slices"
	"strings"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/flags"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/superchain"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	openum "github.com/ethereum-optimism/optimism/op-service/enum"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const EnvVarPrefix = "OP_CHALLENGER"

func prefixEnvVars(names ...string) []string {

	envs := make([]string, 0, len(names))
	for _, name := range names {
		envs = append(envs, EnvVarPrefix+"_"+name)
	}
	return envs
}

var (
	faultDisputeVMs = []gameTypes.GameType{
		gameTypes.CannonGameType,
		gameTypes.CannonKonaGameType,
		gameTypes.SuperCannonGameType,
		gameTypes.SuperCannonKonaGameType,
	}
	// Required Flags
	L1EthRpcFlag = &cli.StringFlag{
		Name:    "l1-eth-rpc",
		Usage:   "HTTP provider URL for L1.",
		EnvVars: prefixEnvVars("L1_ETH_RPC"),
	}
	L1RPCProviderKind = &cli.GenericFlag{
		Name: "l1-rpc-kind",
		Usage: "The kind of RPC provider, used to inform optimal transactions receipts fetching, and thus reduce costs. Valid options: " +
			openum.EnumString(sources.RPCProviderKinds),
		EnvVars: prefixEnvVars("L1_RPC_KIND"),
		Value: func() *sources.RPCProviderKind {
			out := sources.RPCKindStandard
			return &out
		}(),
	}
	L1BeaconFlag = &cli.StringFlag{
		Name:    "l1-beacon",
		Usage:   "Address of L1 Beacon API endpoint to use",
		EnvVars: prefixEnvVars("L1_BEACON"),
	}
	SuperNodeRpcFlag = &cli.StringFlag{
		Name:    "supernode-rpc",
		Usage:   "Provider URL for supernode roots",
		EnvVars: prefixEnvVars("SUPERNODE_RPC"),
	}
	RollupRpcFlag = &cli.StringFlag{
		Name:    "rollup-rpc",
		Usage:   "HTTP provider URL for the rollup node",
		EnvVars: prefixEnvVars("ROLLUP_RPC"),
	}
	NetworkFlag = &cli.StringSliceFlag{
		Name:    flags.NetworkFlagName,
		Usage:   fmt.Sprintf("Predefined network selection. Available networks: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVars: prefixEnvVars("NETWORK"),
	}
	FactoryAddressFlag = &cli.StringFlag{
		Name:    "game-factory-address",
		Usage:   "Address of the fault game factory contract.",
		EnvVars: prefixEnvVars("GAME_FACTORY_ADDRESS"),
	}
	GameAllowlistFlag = &cli.StringSliceFlag{
		Name: "game-allowlist",
		Usage: "List of Fault Game contract addresses the challenger is allowed to play. " +
			"If empty, the challenger will play all games.",
		EnvVars: prefixEnvVars("GAME_ALLOWLIST"),
	}
	GameTypesFlag = &cli.StringSliceFlag{
		Name:    "game-types",
		Aliases: []string{"trace-type"}, // For backwards compatibility
		Usage:   "The game types to support. Valid options: " + openum.EnumStringer(gameTypes.SupportedGameTypes),
		EnvVars: prefixEnvVars("GAME_TYPES", "TRACE_TYPE"),
		Value:   cli.NewStringSlice(gameTypes.CannonGameType.String(), gameTypes.CannonKonaGameType.String()),
	}
	DatadirFlag = &cli.StringFlag{
		Name:    "datadir",
		Usage:   "Directory to store data generated as part of responding to games",
		EnvVars: prefixEnvVars("DATADIR"),
	}
	// Optional Flags
	MaxConcurrencyFlag = &cli.UintFlag{
		Name:    "max-concurrency",
		Usage:   "Maximum number of threads to use when progressing games",
		EnvVars: prefixEnvVars("MAX_CONCURRENCY"),
		Value:   uint(runtime.NumCPU()),
	}
	L2EthRpcFlag = &cli.StringSliceFlag{
		Name:    "l2-eth-rpc",
		Usage:   "URLs of L2 JSON-RPC endpoints to use (eth and debug namespace required)",
		EnvVars: prefixEnvVars("L2_ETH_RPC"),
	}
	L2ExperimentalEthRpcFlag = &cli.StringFlag{
		Name:    "l2-experimental-eth-rpc",
		Usage:   "L2 Address of L2 JSON-RPC endpoint to use (eth and debug namespace required with execution witness support)  (cannon game type only)",
		EnvVars: prefixEnvVars("L2_EXPERIMENTAL_ETH_RPC"),
	}
	MaxPendingTransactionsFlag = &cli.Uint64Flag{
		Name:    "max-pending-tx",
		Usage:   "The maximum number of pending transactions. 0 for no limit.",
		Value:   config.DefaultMaxPendingTx,
		EnvVars: prefixEnvVars("MAX_PENDING_TX"),
	}
	HTTPPollInterval = &cli.DurationFlag{
		Name:    "http-poll-interval",
		Usage:   "Polling interval for latest-block subscription when using an HTTP RPC provider.",
		EnvVars: prefixEnvVars("HTTP_POLL_INTERVAL"),
		Value:   config.DefaultPollInterval,
	}
	MinUpdateInterval = &cli.DurationFlag{
		Name:    "min-update-interval",
		Usage:   "Minimum time between scheduling update cycles based on the L1 block time.",
		EnvVars: prefixEnvVars("MIN_UPDATE_INTERVAL"),
	}
	AdditionalBondClaimants = &cli.StringSliceFlag{
		Name:    "additional-bond-claimants",
		Usage:   "List of addresses to claim bonds for, in addition to the configured transaction sender",
		EnvVars: prefixEnvVars("ADDITIONAL_BOND_CLAIMANTS"),
	}
	PreStatesURLFlag = NewVMFlag("prestates-url", EnvVarPrefix, faultDisputeVMs, func(name string, envVars []string, gameTypeInfo string) cli.Flag {
		return &cli.StringFlag{
			Name: name,
			Usage: "Base URL to absolute prestates to use when generating trace data. " +
				"Prestates in this directory should be name as <commitment>.bin.gz <commitment>.json.gz or <commitment>.json " +
				gameTypeInfo,
			EnvVars: envVars,
		}
	})
	RollupConfigFlag = NewVMFlag("rollup-config", EnvVarPrefix, faultDisputeVMs, func(name string, envVars []string, gameTypeInfo string) cli.Flag {
		return &cli.StringSliceFlag{
			Name:    name,
			Usage:   "Rollup chain parameters " + gameTypeInfo,
			EnvVars: envVars,
		}
	})
	L2GenesisFlag = NewVMFlag("l2-genesis", EnvVarPrefix, faultDisputeVMs, func(name string, envVars []string, gameTypeInfo string) cli.Flag {
		return &cli.StringSliceFlag{
			Name:    name,
			Usage:   "Paths to the op-geth genesis file " + gameTypeInfo,
			EnvVars: envVars,
		}
	})
	L1GenesisFlag = NewVMFlag("l1-genesis", EnvVarPrefix, faultDisputeVMs, func(name string, envVars []string, gameTypeInfo string) cli.Flag {
		return &cli.StringFlag{
			Name:    name,
			Usage:   "Path to the L1 genesis file. Only required if the L1 is not mainnet, sepolia, holesky, or hoodi.",
			EnvVars: envVars,
		}
	})
	DepsetConfigFlag = NewVMFlag("depset-config", EnvVarPrefix, faultDisputeVMs, func(name string, envVars []string, gameTypeInfo string) cli.Flag {
		return &cli.StringFlag{
			Name:    name,
			Usage:   "Interop dependency set config file " + gameTypeInfo,
			EnvVars: envVars,
		}
	})
	CannonL2CustomFlag = &cli.BoolFlag{
		Name: "cannon-l2-custom",
		Usage: "Notify the op-program host that the L2 chain uses custom config to be loaded via the preimage oracle. " +
			"WARNING: This is incompatible with on-chain testing and must only be used for testing purposes.",
		EnvVars: prefixEnvVars("CANNON_L2_CUSTOM"),
		Value:   false,
		Hidden:  true,
	}
	CannonBinFlag = &cli.StringFlag{
		Name:    "cannon-bin",
		Usage:   "Path to cannon executable to use when generating trace data (cannon game type only)",
		EnvVars: prefixEnvVars("CANNON_BIN"),
	}
	CannonServerFlag = &cli.StringFlag{
		Name:    "cannon-server",
		Usage:   "Path to executable to use as pre-image oracle server when generating trace data (cannon game type only)",
		EnvVars: prefixEnvVars("CANNON_SERVER"),
	}
	CannonPreStateFlag = &cli.StringFlag{
		Name:    "cannon-prestate",
		Usage:   "Path to absolute prestate to use when generating trace data (cannon game type only)",
		EnvVars: prefixEnvVars("CANNON_PRESTATE"),
	}
	CannonSnapshotFreqFlag = &cli.UintFlag{
		Name:    "cannon-snapshot-freq",
		Usage:   "Frequency of cannon snapshots to generate in VM steps (cannon game type only)",
		EnvVars: prefixEnvVars("CANNON_SNAPSHOT_FREQ"),
		Value:   config.DefaultCannonSnapshotFreq,
	}
	CannonInfoFreqFlag = &cli.UintFlag{
		Name:    "cannon-info-freq",
		Usage:   "Frequency of cannon info log messages to generate in VM steps (cannon game type only)",
		EnvVars: prefixEnvVars("CANNON_INFO_FREQ"),
		Value:   config.DefaultCannonInfoFreq,
	}
	CannonKonaServerFlag = &cli.StringFlag{
		Name:    "cannon-kona-server",
		Usage:   "Path to kona executable to use as pre-image oracle server when generating trace data (cannon-kona game type only)",
		EnvVars: prefixEnvVars("CANNON_KONA_SERVER"),
	}
	CannonKonaPreStateFlag = &cli.StringFlag{
		Name:    "cannon-kona-prestate",
		Usage:   "Path to absolute prestate to use when generating trace data (cannon-kona game type only)",
		EnvVars: prefixEnvVars("CANNON_KONA_PRESTATE"),
	}
	CannonKonaL2CustomFlag = &cli.BoolFlag{
		Name: "cannon-kona-l2-custom",
		Usage: "Notify the kona-host that the L2 chain uses custom config to be loaded via the preimage oracle. " +
			"WARNING: This is incompatible with on-chain testing and must only be used for testing purposes.",
		EnvVars: prefixEnvVars("CANNON_KONA_L2_CUSTOM"),
		Value:   false,
		Hidden:  true,
	}
	GameWindowFlag = &cli.DurationFlag{
		Name: "game-window",
		Usage: "The time window which the challenger will look for games to progress and claim bonds. " +
			"This should include a buffer for the challenger to claim bonds for games outside the maximum game duration.",
		EnvVars: prefixEnvVars("GAME_WINDOW"),
		Value:   config.DefaultGameWindow,
	}
	SelectiveClaimResolutionFlag = &cli.BoolFlag{
		Name:    "selective-claim-resolution",
		Usage:   "Only resolve claims for the configured claimants",
		EnvVars: prefixEnvVars("SELECTIVE_CLAIM_RESOLUTION"),
	}
	UnsafeAllowInvalidPrestate = &cli.BoolFlag{
		Name:    "unsafe-allow-invalid-prestate",
		Usage:   "Allow responding to games where the absolute prestate is configured incorrectly. THIS IS UNSAFE!",
		EnvVars: prefixEnvVars("UNSAFE_ALLOW_INVALID_PRESTATE"),
		Hidden:  true, // Hidden as this is an unsafe flag added only for testing purposes
	}
	ResponseDelayFlag = &cli.DurationFlag{
		Name:    "response-delay",
		Usage:   "Delay before responding to game actions to slow down game progression.",
		EnvVars: prefixEnvVars("RESPONSE_DELAY"),
		Value:   config.DefaultResponseDelay,
	}
	ResponseDelayAfterFlag = &cli.Uint64Flag{
		Name:    "response-delay-after",
		Usage:   "Number of responses after which to start applying the delay (0 = from first response).",
		EnvVars: prefixEnvVars("RESPONSE_DELAY_AFTER"),
		Value:   config.DefaultResponseDelayAfter,
	}
)

// requiredFlags are checked by [CheckRequired]
var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	DatadirFlag,
	L1BeaconFlag,
}

// optionalFlags is a list of unchecked cli flags
var optionalFlags = []cli.Flag{
	L1RPCProviderKind,
	RollupRpcFlag,
	NetworkFlag,
	FactoryAddressFlag,
	GameTypesFlag,
	MaxConcurrencyFlag,
	SuperNodeRpcFlag,
	L2EthRpcFlag,
	L2ExperimentalEthRpcFlag,
	MaxPendingTransactionsFlag,
	HTTPPollInterval,
	MinUpdateInterval,
	AdditionalBondClaimants,
	GameAllowlistFlag,
	CannonL2CustomFlag,
	CannonBinFlag,
	CannonServerFlag,
	CannonPreStateFlag,
	CannonSnapshotFreqFlag,
	CannonInfoFreqFlag,
	CannonKonaServerFlag,
	CannonKonaPreStateFlag,
	CannonKonaL2CustomFlag,
	GameWindowFlag,
	SelectiveClaimResolutionFlag,
	UnsafeAllowInvalidPrestate,
	ResponseDelayFlag,
	ResponseDelayAfterFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, PreStatesURLFlag.Flags()...)
	optionalFlags = append(optionalFlags, RollupConfigFlag.Flags()...)
	optionalFlags = append(optionalFlags, L1GenesisFlag.Flags()...)
	optionalFlags = append(optionalFlags, L2GenesisFlag.Flags()...)
	optionalFlags = append(optionalFlags, DepsetConfigFlag.Flags()...)
	optionalFlags = append(optionalFlags, txmgr.CLIFlagsWithDefaults(EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func checkOutputProviderFlags(ctx *cli.Context) error {
	if !ctx.IsSet(RollupRpcFlag.Name) {
		return fmt.Errorf("flag %v is required", RollupRpcFlag.Name)
	}
	return nil
}

func CheckCannonBaseFlags(ctx *cli.Context) error {
	if ctx.IsSet(flags.NetworkFlagName) &&
		(RollupConfigFlag.IsSet(ctx, gameTypes.CannonGameType) || L2GenesisFlag.IsSet(ctx, gameTypes.CannonGameType) || L1GenesisFlag.IsSet(ctx, gameTypes.CannonGameType) || ctx.Bool(CannonL2CustomFlag.Name)) {
		return fmt.Errorf("flag %v can not be used with %v, %v, %v or %v",
			flags.NetworkFlagName, RollupConfigFlag.EitherFlagName(gameTypes.CannonGameType), L2GenesisFlag.EitherFlagName(gameTypes.CannonGameType), L1GenesisFlag.EitherFlagName(gameTypes.CannonGameType), CannonL2CustomFlag.Name)
	}
	if ctx.Bool(CannonL2CustomFlag.Name) && !(RollupConfigFlag.IsSet(ctx, gameTypes.CannonGameType) && L2GenesisFlag.IsSet(ctx, gameTypes.CannonGameType)) {
		return fmt.Errorf("flag %v and %v must be set when %v is true",
			RollupConfigFlag.EitherFlagName(gameTypes.CannonGameType), L2GenesisFlag.EitherFlagName(gameTypes.CannonGameType), CannonL2CustomFlag.Name)
	}
	if !ctx.IsSet(CannonBinFlag.Name) {
		return fmt.Errorf("flag %s is required", CannonBinFlag.Name)
	}
	if !ctx.IsSet(CannonServerFlag.Name) {
		return fmt.Errorf("flag %s is required", CannonServerFlag.Name)
	}
	if !PreStatesURLFlag.IsSet(ctx, gameTypes.CannonGameType) && !ctx.IsSet(CannonPreStateFlag.Name) {
		return fmt.Errorf("flag %s or %s is required", PreStatesURLFlag.EitherFlagName(gameTypes.CannonGameType), CannonPreStateFlag.Name)
	}
	return nil
}

func CheckSuperCannonFlags(ctx *cli.Context) error {
	if !ctx.IsSet(SuperNodeRpcFlag.Name) {
		return fmt.Errorf("flag %v is required", SuperNodeRpcFlag.Name)
	}
	if !ctx.IsSet(flags.NetworkFlagName) &&
		!(RollupConfigFlag.IsSet(ctx, gameTypes.CannonGameType) && L2GenesisFlag.IsSet(ctx, gameTypes.CannonGameType) && DepsetConfigFlag.IsSet(ctx, gameTypes.CannonGameType)) {
		return fmt.Errorf("flag %v or %v, %v and %v is required",
			flags.NetworkFlagName,
			RollupConfigFlag.EitherFlagName(gameTypes.CannonGameType),
			L2GenesisFlag.EitherFlagName(gameTypes.CannonGameType),
			DepsetConfigFlag.EitherFlagName(gameTypes.CannonGameType))
	}
	if err := CheckCannonBaseFlags(ctx); err != nil {
		return err
	}
	return nil
}

func CheckSuperCannonKonaFlags(ctx *cli.Context) error {
	if !ctx.IsSet(SuperNodeRpcFlag.Name) {
		return fmt.Errorf("flag %v is required", SuperNodeRpcFlag.Name)
	}
	if !ctx.IsSet(flags.NetworkFlagName) &&
		!(RollupConfigFlag.IsSet(ctx, gameTypes.CannonKonaGameType) && L2GenesisFlag.IsSet(ctx, gameTypes.CannonKonaGameType) && DepsetConfigFlag.IsSet(ctx, gameTypes.CannonKonaGameType)) {
		return fmt.Errorf("flag %v or %v, %v and %v is required",
			flags.NetworkFlagName,
			RollupConfigFlag.EitherFlagName(gameTypes.CannonKonaGameType),
			L2GenesisFlag.EitherFlagName(gameTypes.CannonKonaGameType),
			DepsetConfigFlag.EitherFlagName(gameTypes.CannonKonaGameType))
	}
	if err := CheckCannonKonaBaseFlags(ctx, gameTypes.CannonKonaGameType); err != nil {
		return err
	}
	return nil
}

func CheckCannonFlags(ctx *cli.Context) error {
	if err := checkOutputProviderFlags(ctx); err != nil {
		return err
	}
	if !ctx.IsSet(flags.NetworkFlagName) &&
		!(RollupConfigFlag.IsSet(ctx, gameTypes.CannonGameType) && L2GenesisFlag.IsSet(ctx, gameTypes.CannonGameType)) {
		return fmt.Errorf("flag %v or %v and %v is required",
			flags.NetworkFlagName, RollupConfigFlag.EitherFlagName(gameTypes.CannonGameType), L2GenesisFlag.EitherFlagName(gameTypes.CannonGameType))
	}
	if err := CheckCannonBaseFlags(ctx); err != nil {
		return err
	}
	return nil
}

func CheckCannonKonaBaseFlags(ctx *cli.Context, gameType gameTypes.GameType) error {
	if !ctx.IsSet(flags.NetworkFlagName) &&
		!(RollupConfigFlag.IsSet(ctx, gameType) && L2GenesisFlag.IsSet(ctx, gameType)) {
		return fmt.Errorf("flag %v or %v and %v is required",
			flags.NetworkFlagName, RollupConfigFlag.EitherFlagName(gameType), L2GenesisFlag.EitherFlagName(gameType))
	}
	if ctx.IsSet(flags.NetworkFlagName) &&
		(RollupConfigFlag.IsSet(ctx, gameTypes.CannonKonaGameType) || L2GenesisFlag.IsSet(ctx, gameTypes.CannonKonaGameType) || L1GenesisFlag.IsSet(ctx, gameTypes.CannonKonaGameType) || ctx.Bool(CannonKonaL2CustomFlag.Name)) {
		return fmt.Errorf("flag %v can not be used with %v, %v, %v or %v",
			flags.NetworkFlagName, RollupConfigFlag.EitherFlagName(gameTypes.CannonKonaGameType), L2GenesisFlag.EitherFlagName(gameTypes.CannonKonaGameType), L1GenesisFlag.EitherFlagName(gameTypes.CannonKonaGameType), CannonKonaL2CustomFlag.Name)
	}
	if !ctx.IsSet(CannonBinFlag.Name) {
		return fmt.Errorf("flag %s is required", CannonBinFlag.Name)
	}
	return nil
}

func CheckCannonKonaFlags(ctx *cli.Context) error {
	if err := checkOutputProviderFlags(ctx); err != nil {
		return err
	}
	if err := CheckCannonKonaBaseFlags(ctx, gameTypes.CannonKonaGameType); err != nil {
		return err
	}
	if !ctx.IsSet(CannonKonaServerFlag.Name) {
		return fmt.Errorf("flag %s is required", CannonKonaServerFlag.Name)
	}
	if !PreStatesURLFlag.IsSet(ctx, gameTypes.CannonKonaGameType) && !ctx.IsSet(CannonKonaPreStateFlag.Name) {
		return fmt.Errorf("flag %s or %s is required", PreStatesURLFlag.EitherFlagName(gameTypes.CannonKonaGameType), CannonKonaPreStateFlag.Name)
	}
	return nil
}

func CheckRequired(ctx *cli.Context, types []gameTypes.GameType) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	if !ctx.IsSet(L2EthRpcFlag.Name) {
		return fmt.Errorf("flag %s is required", L2EthRpcFlag.Name)
	}
	for _, gameType := range types {
		switch gameType {
		case gameTypes.CannonGameType, gameTypes.PermissionedGameType:
			if err := CheckCannonFlags(ctx); err != nil {
				return err
			}
		case gameTypes.CannonKonaGameType:
			if err := CheckCannonKonaFlags(ctx); err != nil {
				return err
			}
		case gameTypes.SuperCannonGameType, gameTypes.SuperPermissionedGameType:
			if err := CheckSuperCannonFlags(ctx); err != nil {
				return err
			}
		case gameTypes.SuperCannonKonaGameType:
			if err := CheckSuperCannonKonaFlags(ctx); err != nil {
				return err
			}
		case gameTypes.OptimisticZKGameType, gameTypes.AlphabetGameType, gameTypes.FastGameType:
			if err := checkOutputProviderFlags(ctx); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid game type %v. must be one of %v", gameType, gameTypes.SupportedGameTypes)
		}
	}
	return nil
}

func parseGameTypes(ctx *cli.Context) ([]gameTypes.GameType, error) {
	var result []gameTypes.GameType
	for _, typeName := range ctx.StringSlice(GameTypesFlag.Name) {
		gameType, err := gameTypes.SupportedGameTypeFromString(typeName)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(result, gameType) {
			result = append(result, gameType)
		}
	}
	return result, nil
}

type ChainAddressesSource func(network string) (superchain.AddressesConfig, error)

var superchainAddressSource ChainAddressesSource = func(network string) (superchain.AddressesConfig, error) {
	chainCfg := chaincfg.ChainByName(network)
	if chainCfg == nil {
		return superchain.AddressesConfig{}, fmt.Errorf("unknown chain: %v (Valid options: %v)", network, strings.Join(chaincfg.AvailableNetworks(), ", "))
	}
	return chainCfg.Addresses, nil
}

func FactoryAddress(ctx *cli.Context) (common.Address, error) {
	// Use FactoryAddressFlag in preference to Network. Allows overriding the default dispute game factory.
	if ctx.IsSet(FactoryAddressFlag.Name) {
		gameFactoryAddress, err := opservice.ParseAddress(ctx.String(FactoryAddressFlag.Name))
		if err != nil {
			return common.Address{}, err
		}
		return gameFactoryAddress, nil
	}
	networks := ctx.StringSlice(flags.NetworkFlagName)
	if len(networks) == 0 {
		return common.Address{}, fmt.Errorf("flag %v or %v is required", FactoryAddressFlag.Name, flags.NetworkFlagName)
	}
	return FactoryAddressForNetworks(networks, superchainAddressSource)
}

func FactoryAddressForNetworks(networks []string, addressSource ChainAddressesSource) (common.Address, error) {
	var factoryAddress common.Address
	for _, network := range networks {
		addrs, err := addressSource(network)
		if err != nil {
			return common.Address{}, err
		}
		if addrs.DisputeGameFactoryProxy == nil {
			return common.Address{}, fmt.Errorf("dispute factory proxy not available for chain %v", network)
		}
		addr := *addrs.DisputeGameFactoryProxy
		if factoryAddress == (common.Address{}) {
			factoryAddress = addr
		} else if factoryAddress != addr {
			return common.Address{}, fmt.Errorf("specified networks use different dispute game factories, flag %v required", FactoryAddressFlag.Name)
		}
	}
	return factoryAddress, nil
}

// NewConfigFromCLI parses the Config from the provided flags or environment variables.
func NewConfigFromCLI(ctx *cli.Context, logger log.Logger) (*config.Config, error) {
	enabledGameTypes, err := parseGameTypes(ctx)
	if err != nil {
		return nil, err
	}
	if err := CheckRequired(ctx, enabledGameTypes); err != nil {
		return nil, err
	}
	gameFactoryAddress, err := FactoryAddress(ctx)
	if err != nil {
		return nil, err
	}
	var allowedGames []common.Address
	if ctx.StringSlice(GameAllowlistFlag.Name) != nil {
		for _, addr := range ctx.StringSlice(GameAllowlistFlag.Name) {
			gameAddress, err := opservice.ParseAddress(addr)
			if err != nil {
				return nil, err
			}
			allowedGames = append(allowedGames, gameAddress)
		}
	}

	txMgrConfig := txmgr.ReadCLIConfig(ctx)
	metricsConfig := opmetrics.ReadCLIConfig(ctx)
	pprofConfig := oppprof.ReadCLIConfig(ctx)

	maxConcurrency := ctx.Uint(MaxConcurrencyFlag.Name)
	if maxConcurrency == 0 {
		return nil, fmt.Errorf("%v must not be 0", MaxConcurrencyFlag.Name)
	}
	var claimants []common.Address
	if ctx.IsSet(AdditionalBondClaimants.Name) {
		for _, addrStr := range ctx.StringSlice(AdditionalBondClaimants.Name) {
			claimant, err := opservice.ParseAddress(addrStr)
			if err != nil {
				return nil, fmt.Errorf("invalid additional claimant: %w", err)
			}
			claimants = append(claimants, claimant)
		}
	}

	getPrestatesUrl := func(gameType gameTypes.GameType) (*url.URL, error) {
		var preStatesURL *url.URL
		if PreStatesURLFlag.IsSet(ctx, gameType) {
			val := PreStatesURLFlag.String(ctx, gameType)
			preStatesURL, err = url.Parse(val)
			if err != nil {
				return nil, fmt.Errorf("invalid %v (%v): %w", PreStatesURLFlag.SourceFlagName(ctx, gameType), val, err)
			}
		}
		return preStatesURL, nil
	}
	cannonPreStatesURL, err := getPrestatesUrl(gameTypes.CannonGameType)
	if err != nil {
		return nil, err
	}
	cannonKonaPreStatesURL, err := getPrestatesUrl(gameTypes.CannonKonaGameType)
	if err != nil {
		return nil, err
	}
	networks := ctx.StringSlice(flags.NetworkFlagName)
	l1EthRpc := ctx.String(L1EthRpcFlag.Name)
	l1Beacon := ctx.String(L1BeaconFlag.Name)
	l2Rpcs := ctx.StringSlice(L2EthRpcFlag.Name)
	l2Experimental := ctx.String(L2ExperimentalEthRpcFlag.Name)
	return &config.Config{
		// Required Flags
		L1EthRpc:                l1EthRpc,
		L1RPCKind:               sources.RPCProviderKind(strings.ToLower(ctx.String(L1RPCProviderKind.Name))),
		L1Beacon:                l1Beacon,
		GameTypes:               enabledGameTypes,
		GameFactoryAddress:      gameFactoryAddress,
		GameAllowlist:           allowedGames,
		GameWindow:              ctx.Duration(GameWindowFlag.Name),
		MaxConcurrency:          maxConcurrency,
		L2Rpcs:                  l2Rpcs,
		MaxPendingTx:            ctx.Uint64(MaxPendingTransactionsFlag.Name),
		PollInterval:            ctx.Duration(HTTPPollInterval.Name),
		MinUpdateInterval:       ctx.Duration(MinUpdateInterval.Name),
		AdditionalBondClaimants: claimants,
		RollupRpc:               ctx.String(RollupRpcFlag.Name),
		SuperRPC:                ctx.String(SuperNodeRpcFlag.Name),
		Cannon: vm.Config{
			VmType:            gameTypes.CannonGameType,
			L1:                l1EthRpc,
			L1Beacon:          l1Beacon,
			L2s:               l2Rpcs,
			L2Experimental:    l2Experimental,
			VmBin:             ctx.String(CannonBinFlag.Name),
			Server:            ctx.String(CannonServerFlag.Name),
			Networks:          networks,
			L2Custom:          ctx.Bool(CannonL2CustomFlag.Name),
			RollupConfigPaths: RollupConfigFlag.StringSlice(ctx, gameTypes.CannonGameType),
			L1GenesisPath:     L1GenesisFlag.String(ctx, gameTypes.CannonGameType),
			L2GenesisPaths:    L2GenesisFlag.StringSlice(ctx, gameTypes.CannonGameType),
			DepsetConfigPath:  DepsetConfigFlag.String(ctx, gameTypes.CannonGameType),
			SnapshotFreq:      ctx.Uint(CannonSnapshotFreqFlag.Name),
			InfoFreq:          ctx.Uint(CannonInfoFreqFlag.Name),
			DebugInfo:         true,
			BinarySnapshots:   true,
		},
		CannonAbsolutePreState:        ctx.String(CannonPreStateFlag.Name),
		CannonAbsolutePreStateBaseURL: cannonPreStatesURL,
		CannonKona: vm.Config{
			VmType:            gameTypes.CannonKonaGameType,
			L1:                l1EthRpc,
			L1Beacon:          l1Beacon,
			L2s:               l2Rpcs,
			L2Experimental:    l2Experimental,
			VmBin:             ctx.String(CannonBinFlag.Name),
			Server:            ctx.String(CannonKonaServerFlag.Name),
			Networks:          networks,
			L2Custom:          ctx.Bool(CannonKonaL2CustomFlag.Name),
			RollupConfigPaths: RollupConfigFlag.StringSlice(ctx, gameTypes.CannonKonaGameType),
			L1GenesisPath:     L1GenesisFlag.String(ctx, gameTypes.CannonKonaGameType),
			L2GenesisPaths:    L2GenesisFlag.StringSlice(ctx, gameTypes.CannonKonaGameType),
			DepsetConfigPath:  DepsetConfigFlag.String(ctx, gameTypes.CannonKonaGameType),
			SnapshotFreq:      ctx.Uint(CannonSnapshotFreqFlag.Name),
			InfoFreq:          ctx.Uint(CannonInfoFreqFlag.Name),
			DebugInfo:         true,
			BinarySnapshots:   true,
		},
		CannonKonaAbsolutePreState:        ctx.String(CannonKonaPreStateFlag.Name),
		CannonKonaAbsolutePreStateBaseURL: cannonKonaPreStatesURL,
		Datadir:                           ctx.String(DatadirFlag.Name),
		TxMgrConfig:                       txMgrConfig,
		MetricsConfig:                     metricsConfig,
		PprofConfig:                       pprofConfig,
		SelectiveClaimResolution:          ctx.Bool(SelectiveClaimResolutionFlag.Name),
		AllowInvalidPrestate:              ctx.Bool(UnsafeAllowInvalidPrestate.Name),
		ResponseDelay:                     ctx.Duration(ResponseDelayFlag.Name),
		ResponseDelayAfter:                ctx.Uint64(ResponseDelayAfterFlag.Name),
	}, nil
}
