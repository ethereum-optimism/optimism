package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
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
	MigrateInputV2 *MigrateInputV2 `json:"migrateInput,omitempty"`
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

func MigrateCLI(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(lgr.Handler())

	ctx, cancel := context.WithCancel(cliCtx.Context)
	defer cancel()

	l1RPCUrl := cliCtx.String(deployer.L1RPCURLFlag.Name)
	if l1RPCUrl == "" {
		return fmt.Errorf("missing required flag: %s", deployer.L1RPCURLFlag.Name)
	}

	privateKey := cliCtx.String(deployer.PrivateKeyFlag.Name)
	privateKeyECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	l1RPC, err := rpc.Dial(l1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to dial RPC %s: %w", l1RPCUrl, err)
	}
	l1Client := ethclient.NewClient(l1RPC)

	opcmAddr := common.HexToAddress(cliCtx.String(OPCMImplFlag.Name))

	initBondStr := cliCtx.String(InitialBondFlag.Name)
	initBond, ok := new(big.Int).SetString(initBondStr, 10)
	if !ok {
		return fmt.Errorf("failed to parse initial bond: %s", initBondStr)
	}

	input := InteropMigrationInput{
		Prank: common.Address{}, // The current CLI does not support prank address, so we set it to zero.
		Opcm:  opcmAddr,
		MigrateInputV1: &MigrateInputV1{
			UsePermissionlessGame: cliCtx.Bool(PermissionlessFlag.Name),
			StartingAnchorRoot: Proposal{
				Root:             common.HexToHash(cliCtx.String(StartingAnchorRootFlag.Name)),
				L2SequenceNumber: new(big.Int).SetUint64(cliCtx.Uint64(StartingAnchorL2SequenceNumberFlag.Name)),
			},
			GameParameters: GameParameters{
				Proposer:         common.HexToAddress(cliCtx.String(ProposerFlag.Name)),
				Challenger:       common.HexToAddress(cliCtx.String(ChallengerFlag.Name)),
				MaxGameDepth:     cliCtx.Uint64(DisputeMaxGameDepthFlag.Name),
				SplitDepth:       cliCtx.Uint64(DisputeSplitDepthFlag.Name),
				InitBond:         initBond,
				ClockExtension:   cliCtx.Uint64(DisputeClockExtensionFlag.Name),
				MaxClockDuration: cliCtx.Uint64(DisputeMaxClockDurationFlag.Name),
			},
			OpChainConfigs: []OPChainConfig{
				{
					SystemConfigProxy:  common.HexToAddress(cliCtx.String(SystemConfigProxyFlag.Name)),
					CannonPrestate:     common.HexToHash(cliCtx.String(DisputeAbsolutePrestateCannonFlag.Name)),
					CannonKonaPrestate: common.HexToHash(cliCtx.String(DisputeAbsolutePrestateCannonKonaFlag.Name)),
				},
			},
		},
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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("failed to encode interop migration output: %w", err)
	}

	return nil
}

func MigrateCLIV2(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(lgr.Handler())

	ctx, cancel := context.WithCancel(cliCtx.Context)
	defer cancel()

	l1RPCUrl := cliCtx.String(deployer.L1RPCURLFlag.Name)
	if l1RPCUrl == "" {
		return fmt.Errorf("missing required flag: %s", deployer.L1RPCURLFlag.Name)
	}

	privateKey := cliCtx.String(deployer.PrivateKeyFlag.Name)
	privateKeyECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	l1RPC, err := rpc.Dial(l1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to dial RPC %s: %w", l1RPCUrl, err)
	}
	l1Client := ethclient.NewClient(l1RPC)

	opcmAddr := common.HexToAddress(cliCtx.String(OPCMImplFlag.Name))

	initBondStr := cliCtx.String(InitialBondFlag.Name)
	initBond, ok := new(big.Int).SetString(initBondStr, 10)
	if !ok {
		return fmt.Errorf("failed to parse initial bond: %s", initBondStr)
	}

	input := InteropMigrationInput{
		Prank: common.Address{}, // The current CLI does not support prank address, so we set it to zero.
		Opcm:  opcmAddr,
		MigrateInputV2: &MigrateInputV2{
			ChainSystemConfigs: []common.Address{
				common.HexToAddress(cliCtx.String(SystemConfigProxyFlag.Name)),
			},
			DisputeGameConfigs: []DisputeGameConfig{
				{
					Enabled:  cliCtx.Bool(DisputeGameEnabledFlag.Name),
					InitBond: initBond,
					GameType: uint32(cliCtx.Uint64(DisputeGameTypeFlag.Name)),
					GameArgs: common.FromHex(cliCtx.String(DisputeAbsolutePrestateFlag.Name)),
				},
			},
			StartingAnchorRoot: Proposal{
				Root:             common.HexToHash(cliCtx.String(StartingAnchorRootFlag.Name)),
				L2SequenceNumber: new(big.Int).SetUint64(cliCtx.Uint64(StartingAnchorL2SequenceNumberFlag.Name)),
			},
			StartingRespectedGameType: uint32(cliCtx.Uint64(StartingRespectedGameTypeFlag.Name)),
		},
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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("failed to encode interop migration output: %w", err)
	}

	return nil
}
