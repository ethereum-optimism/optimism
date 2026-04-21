package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
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
type ScriptInput struct {
	Prank        common.Address `evm:"prank"`
	Opcm         common.Address `evm:"opcm"`
	MigrateInput []byte         `evm:"migrateInput"`
}

// InteropMigrationInput represents the struct that is read from the config file.
type InteropMigrationInput struct {
	Prank          common.Address  `json:"prank"`
	Opcm           common.Address  `json:"opcm"`
	MigrateInputV2 *MigrateInputV2 `json:"migrateInputV2,omitempty"`
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
type Proposal struct {
	Root             common.Hash `json:"root"`
	L2SequenceNumber *big.Int    `json:"l2SequenceNumber"`
}

// InteropMigrationOutput contains the output of the interop migration script.
type InteropMigrationOutput struct {
	DisputeGameFactory common.Address `json:"disputeGameFactory"`
}

var migrateInputV2Encoder = w3.MustNewFunc(
	"dummy((address[] chainSystemConfigs,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs,(bytes32 root,uint256 l2SequenceNumber) startingAnchorRoot,uint32 startingRespectedGameType))",
	"",
)

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
	if input.MigrateInputV2 == nil {
		return InteropMigrationOutput{}, fmt.Errorf("MigrateInputV2 is required")
	}

	encodedMigrateInput, err := input.EncodedMigrateInputV2()
	if err != nil {
		return InteropMigrationOutput{}, err
	}

	scriptInput := ScriptInput{
		Prank:        input.Prank,
		Opcm:         input.Opcm,
		MigrateInput: encodedMigrateInput,
	}
	return opcm.RunScriptSingle[ScriptInput, InteropMigrationOutput](host, scriptInput, "InteropMigration.s.sol", "InteropMigration")
}

// MigrateCLI is the main function for the migrate command. It validates required flags and runs the migration.
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
	if privateKey == "" {
		return fmt.Errorf("missing required flag: %s", deployer.PrivateKeyFlag.Name)
	}
	privateKeyECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	opcmFlag := cliCtx.String(OPCMImplFlag.Name)
	if opcmFlag == "" {
		return fmt.Errorf("missing required flag: %s", OPCMImplFlag.Name)
	}
	opcmAddr := common.HexToAddress(opcmFlag)

	systemConfigProxyFlag := cliCtx.String(SystemConfigProxyFlag.Name)
	if systemConfigProxyFlag == "" {
		return fmt.Errorf("missing required flag: %s", SystemConfigProxyFlag.Name)
	}

	startingAnchorRootFlag := cliCtx.String(StartingAnchorRootFlag.Name)
	if startingAnchorRootFlag == "" {
		return fmt.Errorf("missing required flag: %s", StartingAnchorRootFlag.Name)
	}

	initBondStr := cliCtx.String(InitialBondFlag.Name)
	if initBondStr == "" {
		return fmt.Errorf("missing required flag: %s", InitialBondFlag.Name)
	}
	initBond, ok := new(big.Int).SetString(initBondStr, 10)
	if !ok {
		return fmt.Errorf("failed to parse initial bond: %s", initBondStr)
	}

	l1RPC, err := rpc.Dial(l1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to dial RPC %s: %w", l1RPCUrl, err)
	}

	l1ProxyAdminOwnerFlag := cliCtx.String(L1ProxyAdminOwnerFlag.Name)
	if l1ProxyAdminOwnerFlag == "" {
		return fmt.Errorf("missing required flag: %s", L1ProxyAdminOwnerFlag.Name)
	}

	disputeAbsolutePrestateFlag := cliCtx.String(DisputeAbsolutePrestateFlag.Name)
	if disputeAbsolutePrestateFlag == "" {
		return fmt.Errorf("missing required flag: %s", DisputeAbsolutePrestateFlag.Name)
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

	absolutePrestate := common.HexToHash(disputeAbsolutePrestateFlag)

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return fmt.Errorf("failed to create bytes32 ABI type: %w", err)
	}

	gameArgs, err := abi.Arguments{{Type: bytes32Type}}.Pack(absolutePrestate)
	if err != nil {
		return fmt.Errorf("failed to ABI-encode game args: %w", err)
	}

	input := InteropMigrationInput{
		Prank: common.HexToAddress(l1ProxyAdminOwnerFlag),
		Opcm:  opcmAddr,
		MigrateInputV2: &MigrateInputV2{
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

	l1Client := ethclient.NewClient(l1RPC)
	defer l1Client.Close()

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
