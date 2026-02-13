package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/urfave/cli/v2"
)

// ScriptInput represents the input struct that is actually passed to the script.
// It contains the prank address, OPCM address, and ABI-encoded migrate input.
// The migrateInput field contains either encoded MigrateInputV1 or MigrateInputV2.
type ScriptInput struct {
	Prank        common.Address `evm:"prank"`
	Opcm         common.Address `evm:"opcm"`
	MigrateInput []byte         `evm:"migrateInput"`
}

// InteropMigrationInput represents the struct that is read from the config file.
// It contains both fields for the old and new migrate input to support both OPCM v1 and v2.
// Only one of MigrateInputV1 or MigrateInputV2 should be set.
type InteropMigrationInput struct {
	Prank          common.Address  `json:"prank"`
	Opcm           common.Address  `json:"opcm"`
	MigrateInputV1 *MigrateInputV1 `json:"migrateInputV1,omitempty"`
	MigrateInputV2 *MigrateInputV2 `json:"migrateInputV2,omitempty"`
}

// MigrateInputV1 represents the migrate input format for OPCM v1 (< 7.0.0).
// This format is used for the interop migration on chains using older OPCM versions.
// Corresponds to IOPContractsManagerInteropMigrator.MigrateInput
type MigrateInputV1 struct {
	UsePermissionlessGame bool            `json:"usePermissionlessGame"`
	StartingAnchorRoot    Proposal        `json:"startingAnchorRoot"`
	GameParameters        GameParameters  `json:"gameParameters"`
	OpChainConfigs        []OPChainConfig `json:"opChainConfigs"`
}

// GameParameters defines the configuration parameters for the fault dispute game.
// Corresponds to IOPContractsManagerInteropMigrator.GameParameters
type GameParameters struct {
	Proposer         common.Address `json:"proposer"`
	Challenger       common.Address `json:"challenger"`
	MaxGameDepth     uint64         `json:"maxGameDepth"`
	SplitDepth       uint64         `json:"splitDepth"`
	InitBond         *big.Int       `json:"initBond"`
	ClockExtension   uint64         `json:"clockExtension"`
	MaxClockDuration uint64         `json:"maxClockDuration"`
}

// OPChainConfig contains per-chain configuration for OPCM v1 migrations.
// Corresponds to IOPContractsManagerInteropMigrator.OPChainConfig
type OPChainConfig struct {
	SystemConfigProxy  common.Address `json:"systemConfigProxy"`
	CannonPrestate     common.Hash    `json:"cannonPrestate"`
	CannonKonaPrestate common.Hash    `json:"cannonKonaPrestate"`
}

// MigrateInputV2 represents the migrate input format for OPCM v2 (>= 7.0.0).
// Corresponds to IOPContractsManagerMigrator.MigrateInput
type MigrateInputV2 struct {
	ChainSystemConfigs        []common.Address    `json:"chainSystemConfigs"`
	DisputeGameConfigs        []DisputeGameConfig `json:"disputeGameConfigs"`
	StartingAnchorRoot        Proposal            `json:"startingAnchorRoot"`
	StartingRespectedGameType uint32              `json:"startingRespectedGameType"`
}

// DisputeGameConfig defines the configuration for a specific dispute game type.
// Corresponds to IOPContractsManagerMigrator.DisputeGameConfig
type DisputeGameConfig struct {
	Enabled  bool     `json:"enabled"`
	InitBond *big.Int `json:"initBond"`
	GameType uint32   `json:"gameType"`
	GameArgs []byte   `json:"gameArgs"`
}

// Proposal represents an L2 output root proposal used as the starting anchor for dispute games.
// Both present in MigrateInputV1 and MigrateInputV2.
type Proposal struct {
	Root             common.Hash `json:"root"`
	L2SequenceNumber *big.Int    `json:"l2SequenceNumber"`
}

// InteropMigrationOutput contains the output of the interop migration script.
type InteropMigrationOutput struct {
	DisputeGameFactory common.Address `json:"disputeGameFactory"`
}

// ABI encoders for migrate inputs
// Note: Duration is uint64 in Solidity but we encode as uint256 since w3 doesn't support uint64
// This works because ABI encoding pads all uints to 32 bytes, and our values fit in uint64
var migrateInputV1Encoder = w3.MustNewFunc(
	"dummy((bool usePermissionlessGame,(bytes32 root,uint256 l2SequenceNumber) startingAnchorRoot,(address proposer,address challenger,uint256 maxGameDepth,uint256 splitDepth,uint256 initBond,uint256 clockExtension,uint256 maxClockDuration) gameParameters,(address systemConfigProxy,bytes32 cannonPrestate,bytes32 cannonKonaPrestate)[] opChainConfigs))",
	"",
)

var migrateInputV2Encoder = w3.MustNewFunc(
	"dummy((address[] chainSystemConfigs,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs,(bytes32 root,uint256 l2SequenceNumber) startingAnchorRoot,uint32 startingRespectedGameType))",
	"",
)

func (i *InteropMigrationInput) EncodedMigrateInputV1() ([]byte, error) {
	if i.MigrateInputV1 == nil {
		return nil, fmt.Errorf("MigrateInputV1 is nil")
	}

	// Convert uint64 fields to *big.Int for encoding since w3 doesn't support uint64
	encodableInput := struct {
		UsePermissionlessGame bool
		StartingAnchorRoot    Proposal
		GameParameters        struct {
			Proposer         common.Address
			Challenger       common.Address
			MaxGameDepth     *big.Int
			SplitDepth       *big.Int
			InitBond         *big.Int
			ClockExtension   *big.Int
			MaxClockDuration *big.Int
		}
		OpChainConfigs []OPChainConfig
	}{
		UsePermissionlessGame: i.MigrateInputV1.UsePermissionlessGame,
		StartingAnchorRoot:    i.MigrateInputV1.StartingAnchorRoot,
		OpChainConfigs:        i.MigrateInputV1.OpChainConfigs,
	}

	encodableInput.GameParameters.Proposer = i.MigrateInputV1.GameParameters.Proposer
	encodableInput.GameParameters.Challenger = i.MigrateInputV1.GameParameters.Challenger
	encodableInput.GameParameters.MaxGameDepth = new(big.Int).SetUint64(i.MigrateInputV1.GameParameters.MaxGameDepth)
	encodableInput.GameParameters.SplitDepth = new(big.Int).SetUint64(i.MigrateInputV1.GameParameters.SplitDepth)
	encodableInput.GameParameters.InitBond = i.MigrateInputV1.GameParameters.InitBond
	encodableInput.GameParameters.ClockExtension = new(big.Int).SetUint64(i.MigrateInputV1.GameParameters.ClockExtension)
	encodableInput.GameParameters.MaxClockDuration = new(big.Int).SetUint64(i.MigrateInputV1.GameParameters.MaxClockDuration)

	data, err := migrateInputV1Encoder.EncodeArgs(encodableInput)
	if err != nil {
		return nil, fmt.Errorf("failed to encode migrate input v1: %w", err)
	}

	if len(data) < 4 {
		return nil, fmt.Errorf("failed to encode migrate input v1: data is too short")
	}

	// Skip the function selector (first 4 bytes)
	return data[4:], nil
}

func (i *InteropMigrationInput) EncodedMigrateInputV2() ([]byte, error) {
	if i.MigrateInputV2 == nil {
		return nil, fmt.Errorf("MigrateInputV2 is nil")
	}
	data, err := migrateInputV2Encoder.EncodeArgs(i.MigrateInputV2)
	if err != nil {
		return nil, fmt.Errorf("failed to encode migrate input v2: %w", err)
	}

	if len(data) < 4 {
		return nil, fmt.Errorf("failed to encode migrate input v2: data is too short")
	}

	// Skip the function selector (first 4 bytes)
	return data[4:], nil
}

func (output *InteropMigrationOutput) CheckOutput(input common.Address) error {
	return nil
}

func Migrate(host *script.Host, input InteropMigrationInput) (InteropMigrationOutput, error) {
	// We need to check which of the two versions of the input we are using.
	var encodedMigrateInput []byte
	var encodedError error
	if input.MigrateInputV2 == nil && input.MigrateInputV1 == nil {
		return InteropMigrationOutput{}, fmt.Errorf("failed to read either a migrate input v1 or v2")
	} else if input.MigrateInputV2 != nil {
		encodedMigrateInput, encodedError = input.EncodedMigrateInputV2()
	} else {
		encodedMigrateInput, encodedError = input.EncodedMigrateInputV1()
	}

	if encodedError != nil {
		return InteropMigrationOutput{}, encodedError
	}

	scriptInput := ScriptInput{
		Prank:        input.Prank,
		Opcm:         input.Opcm,
		MigrateInput: encodedMigrateInput,
	}
	return opcm.RunScriptSingle[ScriptInput, InteropMigrationOutput](host, scriptInput, "InteropMigration.s.sol", "InteropMigration")
}

// MigrateCLI is the main function for the migrate command. It validates required flags and runs the migration,
// using the OPCM to determine the input version it should use.
func MigrateCLI(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(lgr.Handler())

	ctx, cancel := context.WithCancel(cliCtx.Context)
	defer cancel()

	// Validate common required flags for both OPCM v1 and v2 before version detection
	l1RPCUrl := cliCtx.String(deployer.L1RPCURLFlag.Name)
	if l1RPCUrl == "" {
		return fmt.Errorf("missing required flag: %s", deployer.L1RPCURLFlag.Name)
	}

	privateKey := cliCtx.String(deployer.PrivateKeyFlag.Name)
	if privateKey == "" {
		return fmt.Errorf("missing required flag: %s", deployer.PrivateKeyFlag.Name)
	}
	privateKeyECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Get OPCM address
	opcmFlag := cliCtx.String(OPCMImplFlag.Name)
	if opcmFlag == "" {
		return fmt.Errorf("missing required flag: %s", OPCMImplFlag.Name)
	}
	opcmAddr := common.HexToAddress(opcmFlag)

	// Get system config proxy address
	systemConfigProxyFlag := cliCtx.String(SystemConfigProxyFlag.Name)
	if systemConfigProxyFlag == "" {
		return fmt.Errorf("missing required flag: %s", SystemConfigProxyFlag.Name)
	}

	// Get starting anchor root
	startingAnchorRootFlag := cliCtx.String(StartingAnchorRootFlag.Name)
	if startingAnchorRootFlag == "" {
		return fmt.Errorf("missing required flag: %s", StartingAnchorRootFlag.Name)
	}

	// Get initial bond
	initBondStr := cliCtx.String(InitialBondFlag.Name)
	if initBondStr == "" {
		return fmt.Errorf("missing required flag: %s", InitialBondFlag.Name)
	}
	initBond, ok := new(big.Int).SetString(initBondStr, 10)
	if !ok {
		return fmt.Errorf("failed to parse initial bond: %s", initBondStr)
	}

	// Get L1 RPC to check OPCM version
	l1RPC, err := rpc.Dial(l1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to dial RPC %s: %w", l1RPCUrl, err)
	}
	l1Client := ethclient.NewClient(l1RPC)
	defer l1Client.Close()

	opcmContract := opcm.NewContract(opcmAddr, l1Client)

	versionStr, err := opcmContract.GenericStringGetter(ctx, "version")
	if err != nil {
		return fmt.Errorf("failed to get OPCM version: %w", err)
	}

	// Parse version string (format: "major.minor.patch")
	// If version < 7.0.0, use v1, otherwise use v2
	isOPCMv2, err := deployer.IsVersionAtLeast(versionStr, 7, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to parse OPCM version %s: %w", versionStr, err)
	}

	// Get L1 proxy admin owner address
	l1ProxyAdminOwnerFlag := cliCtx.String(L1ProxyAdminOwnerFlag.Name)
	if l1ProxyAdminOwnerFlag == "" {
		return fmt.Errorf("missing required flag: %s", L1ProxyAdminOwnerFlag.Name)
	}

	input := InteropMigrationInput{
		Prank: common.HexToAddress(l1ProxyAdminOwnerFlag),
		Opcm:  opcmAddr,
	}

	if isOPCMv2 {
		disputeAbsolutePrestateFlag := cliCtx.String(DisputeAbsolutePrestateFlag.Name)
		if disputeAbsolutePrestateFlag == "" {
			return fmt.Errorf("missing required flag for OPCM v2: %s", DisputeAbsolutePrestateFlag.Name)
		}

		disputeGameTypeU64 := cliCtx.Uint64(DisputeGameTypeFlag.Name)
		if disputeGameTypeU64 > 0xFFFFFFFF {
			return fmt.Errorf("disputeGameType %d exceeds uint32 max value", disputeGameTypeU64)
		}
		disputeGameType := uint32(disputeGameTypeU64)

		migrateStartingRespectedGameTypeU64 := cliCtx.Uint64(MigrateStartingRespectedGameTypeFlag.Name)
		if migrateStartingRespectedGameTypeU64 > 0xFFFFFFFF {
			return fmt.Errorf("startingRespectedGameType %d exceeds uint32 max value", migrateStartingRespectedGameTypeU64)
		}
		migrateStartingRespectedGameType := uint32(migrateStartingRespectedGameTypeU64)

		// ABI-encode the FaultDisputeGameConfig struct
		// FaultDisputeGameConfig contains a single field: absolutePrestate (bytes32)
		absolutePrestateHex := cliCtx.String(DisputeAbsolutePrestateFlag.Name)
		absolutePrestate := common.HexToHash(absolutePrestateHex)

		bytes32Type, err := abi.NewType("bytes32", "", nil)
		if err != nil {
			return fmt.Errorf("failed to create bytes32 ABI type: %w", err)
		}

		gameArgs, err := abi.Arguments{{Type: bytes32Type}}.Pack(absolutePrestate)
		if err != nil {
			return fmt.Errorf("failed to ABI-encode game args: %w", err)
		}

		// V2 Migration Input
		input.MigrateInputV2 = &MigrateInputV2{
			ChainSystemConfigs: []common.Address{
				common.HexToAddress(systemConfigProxyFlag),
			},
			DisputeGameConfigs: []DisputeGameConfig{
				{
					Enabled:  cliCtx.Bool(MigrateDisputeGameEnabledFlag.Name),
					InitBond: initBond,
					GameType: disputeGameType,
					GameArgs: gameArgs,
				},
			},
			StartingAnchorRoot: Proposal{
				Root:             common.HexToHash(startingAnchorRootFlag),
				L2SequenceNumber: new(big.Int).SetUint64(cliCtx.Uint64(StartingAnchorL2SequenceNumberFlag.Name)),
			},
			StartingRespectedGameType: migrateStartingRespectedGameType,
		}
	} else {
		// Validate V1-specific required flags
		proposerFlag := cliCtx.String(ProposerFlag.Name)
		if proposerFlag == "" {
			return fmt.Errorf("missing required flag for OPCM v1: %s", ProposerFlag.Name)
		}

		challengerFlag := cliCtx.String(ChallengerFlag.Name)
		if challengerFlag == "" {
			return fmt.Errorf("missing required flag for OPCM v1: %s", ChallengerFlag.Name)
		}

		cannonPrestateFlag := cliCtx.String(DisputeAbsolutePrestateCannonFlag.Name)
		if cannonPrestateFlag == "" {
			return fmt.Errorf("missing required flag for OPCM v1: %s", DisputeAbsolutePrestateCannonFlag.Name)
		}

		cannonKonaPrestateFlag := cliCtx.String(DisputeAbsolutePrestateCannonKonaFlag.Name)
		if cannonKonaPrestateFlag == "" {
			return fmt.Errorf("missing required flag for OPCM v1: %s", DisputeAbsolutePrestateCannonKonaFlag.Name)
		}

		// V1 Migration Input
		input.MigrateInputV1 = &MigrateInputV1{
			UsePermissionlessGame: cliCtx.Bool(PermissionlessFlag.Name),
			StartingAnchorRoot: Proposal{
				Root:             common.HexToHash(startingAnchorRootFlag),
				L2SequenceNumber: new(big.Int).SetUint64(cliCtx.Uint64(StartingAnchorL2SequenceNumberFlag.Name)),
			},
			GameParameters: GameParameters{
				Proposer:         common.HexToAddress(proposerFlag),
				Challenger:       common.HexToAddress(challengerFlag),
				MaxGameDepth:     cliCtx.Uint64(DisputeMaxGameDepthFlag.Name),
				SplitDepth:       cliCtx.Uint64(DisputeSplitDepthFlag.Name),
				InitBond:         initBond,
				ClockExtension:   cliCtx.Uint64(DisputeClockExtensionFlag.Name),
				MaxClockDuration: cliCtx.Uint64(DisputeMaxClockDurationFlag.Name),
			},
			// At the moment we only support a single chain config
			OpChainConfigs: []OPChainConfig{
				{
					SystemConfigProxy:  common.HexToAddress(systemConfigProxyFlag),
					CannonPrestate:     common.HexToHash(cannonPrestateFlag),
					CannonKonaPrestate: common.HexToHash(cannonKonaPrestateFlag),
				},
			},
		}
	}

	artifactsLocatorStr := cliCtx.String(deployer.ArtifactsLocatorFlag.Name)
	artifactsLocator := new(artifacts.Locator)
	if err := artifactsLocator.UnmarshalText([]byte(artifactsLocatorStr)); err != nil {
		return fmt.Errorf("failed to parse artifacts locator: %w", err)
	}

	cacheDir := cliCtx.String(deployer.CacheDirFlag.Name)
	artifactsFS, err := artifacts.Download(ctx, artifactsLocator, ioutil.BarProgressor(), cacheDir)
	if err != nil {
		return fmt.Errorf("failed to download artifacts: %w", err)
	}

	l1ChainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(privateKeyECDSA, l1ChainID))
	deployerAddr := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
	bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  lgr,
		ChainID: l1ChainID,
		Client:  l1Client,
		Signer:  signer,
		From:    deployerAddr,
	})
	if err != nil {
		return fmt.Errorf("failed to create broadcaster: %w", err)
	}

	l1Host, err := env.DefaultForkedScriptHost(
		ctx,
		bcaster,
		lgr,
		deployerAddr,
		artifactsFS,
		l1RPC,
	)
	if err != nil {
		return fmt.Errorf("failed to create script host: %w", err)
	}

	output, err := Migrate(l1Host, input)
	if err != nil {
		return fmt.Errorf("failed to run interop migration: %w", err)
	}

	enc := json.NewEncoder(cliCtx.App.Writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("failed to encode interop migration output: %w", err)
	}

	return nil
}
