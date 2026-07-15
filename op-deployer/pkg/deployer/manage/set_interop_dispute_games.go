package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/urfave/cli/v2"
)

// gameTypeZKDisputeGame is GameTypes.ZK_DISPUTE_GAME (10) in the Solidity GameTypes library.
const gameTypeZKDisputeGame uint32 = 10

// zkGameArgsEncoder ABI-encodes the ZKDisputeGameConfig consumed by
// OPContractsManagerUtils._makeGameArgs for the ZK dispute game. It mirrors the encoder in the
// embedded upgrader; the five fields match IOPContractsManagerUtils.ZKDisputeGameConfig.
var zkGameArgsEncoder = w3.MustNewFunc(
	"dummy((bytes32 absolutePrestate,address verifier,uint64 maxChallengeDuration,uint64 maxProveDuration,uint256 challengerBond))",
	"",
)

// zkDisputeGameConfig mirrors IOPContractsManagerUtils.ZKDisputeGameConfig.
type zkDisputeGameConfig struct {
	AbsolutePrestate     common.Hash
	Verifier             common.Address
	MaxChallengeDuration uint64
	MaxProveDuration     uint64
	ChallengerBond       *big.Int
}

// encodeZKGameArgs ABI-encodes a ZKDisputeGameConfig into the `gameArgs` bytes expected by the
// DisputeGameConfig entry for the ZK dispute game type.
func encodeZKGameArgs(cfg zkDisputeGameConfig) ([]byte, error) {
	if cfg.Verifier == (common.Address{}) {
		return nil, fmt.Errorf("ZK verifier must not be the zero address")
	}
	if cfg.AbsolutePrestate == (common.Hash{}) {
		return nil, fmt.Errorf("ZK absolutePrestate must not be zero")
	}
	if cfg.MaxChallengeDuration == 0 {
		return nil, fmt.Errorf("ZK maxChallengeDuration must be > 0")
	}
	if cfg.MaxProveDuration == 0 {
		return nil, fmt.Errorf("ZK maxProveDuration must be > 0")
	}
	if cfg.ChallengerBond == nil || cfg.ChallengerBond.Sign() <= 0 {
		return nil, fmt.Errorf("ZK challengerBond must be a positive value")
	}

	data, err := zkGameArgsEncoder.EncodeArgs(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode ZK game args: %w", err)
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("failed to encode ZK game args: data too short")
	}
	// Skip the function selector (first 4 bytes).
	return data[4:], nil
}

// SetInteropDisputeGames runs the SetInteropDisputeGames forge script, which swaps the
// shared dispute games of an already-interop set to a new super game (e.g. ZKDisputeGame). It
// reuses the migrate input types since the on-chain entry point takes the same MigrateInput.
func SetInteropDisputeGames(host *script.Host, input InteropMigrationInput) (InteropMigrationOutput, error) {
	if input.MigrateInputV2 == nil {
		return InteropMigrationOutput{}, fmt.Errorf("MigrateInputV2 is required")
	}

	encoded, err := input.EncodedMigrateInputV2()
	if err != nil {
		return InteropMigrationOutput{}, err
	}

	scriptInput := ScriptInput{
		Prank:        input.Prank,
		Opcm:         input.Opcm,
		MigrateInput: encoded,
	}
	return opcm.RunScriptSingle[ScriptInput, InteropMigrationOutput](
		host, scriptInput, "SetInteropDisputeGames.s.sol", "SetInteropDisputeGames",
	)
}

// SetInteropDisputeGamesCLI is the entry point for the `manage set-interop-dispute-games`
// command. It builds the swap input (disable the source super game, enable the ZK dispute game,
// set the respected game type to ZK) and runs the swap against a forked L1.
func SetInteropDisputeGamesCLI(cliCtx *cli.Context) error {
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
	if !common.IsHexAddress(opcmFlag) {
		return fmt.Errorf("flag %s must be a valid address, got %q", OPCMImplFlag.Name, opcmFlag)
	}
	opcmAddr := common.HexToAddress(opcmFlag)

	l1ProxyAdminOwnerFlag := cliCtx.String(L1ProxyAdminOwnerFlag.Name)
	if !common.IsHexAddress(l1ProxyAdminOwnerFlag) {
		return fmt.Errorf("flag %s must be a valid address, got %q", L1ProxyAdminOwnerFlag.Name, l1ProxyAdminOwnerFlag)
	}

	// The interop set: a comma-separated list of SystemConfig proxy addresses. Validate each one
	// up front — HexToAddress silently coerces malformed input, so a typo could otherwise mutate
	// the wrong contracts.
	chainsFlag := cliCtx.String(SystemConfigProxyAddressesFlag.Name)
	if chainsFlag == "" {
		return fmt.Errorf("missing required flag: %s", SystemConfigProxyAddressesFlag.Name)
	}
	var chainSystemConfigs []common.Address
	for _, raw := range strings.Split(chainsFlag, ",") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if !common.IsHexAddress(trimmed) {
			return fmt.Errorf("flag %s contains an invalid address: %q", SystemConfigProxyAddressesFlag.Name, trimmed)
		}
		chainSystemConfigs = append(chainSystemConfigs, common.HexToAddress(trimmed))
	}
	if len(chainSystemConfigs) == 0 {
		return fmt.Errorf("flag %s must contain at least one address", SystemConfigProxyAddressesFlag.Name)
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

	// Source super games to retire (cleared by the swap). By default only SUPER_CANNON_KONA (9) is
	// cleared — the permissionless super fault game that ZK replaces. SUPER_PERMISSIONED (5)
	// is deliberately kept as a permissioned liveness backup (in case the prover/ZK path stalls),
	// mirroring today's SPDG + permissionless shape. A disabled config for a type that is not
	// registered is a harmless no-op on-chain.
	var disputeGameConfigs []DisputeGameConfig
	for _, raw := range strings.Split(cliCtx.String(SourceGameTypesFlag.Name), ",") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		gt, err := strconv.ParseUint(trimmed, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid source game type %q: %w", trimmed, err)
		}
		if uint32(gt) == gameTypeZKDisputeGame {
			return fmt.Errorf(
				"--%s must not contain the ZK dispute game type (%d)", SourceGameTypesFlag.Name, gameTypeZKDisputeGame,
			)
		}
		disputeGameConfigs = append(disputeGameConfigs, DisputeGameConfig{
			Enabled:  false,
			InitBond: big.NewInt(0),
			GameType: uint32(gt),
			GameArgs: []byte{},
		})
	}

	// Encode the ZK dispute game args and enable it.
	verifierFlag := cliCtx.String(ZKVerifierFlag.Name)
	if !common.IsHexAddress(verifierFlag) {
		return fmt.Errorf("flag %s must be a valid address, got %q", ZKVerifierFlag.Name, verifierFlag)
	}
	if !cliCtx.IsSet(DisputeAbsolutePrestateFlag.Name) {
		return fmt.Errorf("flag %s must be explicitly set for the ZK dispute game; its default is the cannon/kona prestate which is invalid for ZK", DisputeAbsolutePrestateFlag.Name)
	}
	zkArgs, err := encodeZKGameArgs(zkDisputeGameConfig{
		AbsolutePrestate:     common.HexToHash(cliCtx.String(DisputeAbsolutePrestateFlag.Name)),
		Verifier:             common.HexToAddress(verifierFlag),
		MaxChallengeDuration: cliCtx.Uint64(ZKMaxChallengeDurationFlag.Name),
		MaxProveDuration:     cliCtx.Uint64(ZKMaxProveDurationFlag.Name),
		ChallengerBond:       initBond,
	})
	if err != nil {
		return err
	}
	disputeGameConfigs = append(disputeGameConfigs, DisputeGameConfig{
		Enabled:  true,
		InitBond: initBond,
		GameType: gameTypeZKDisputeGame,
		GameArgs: zkArgs,
	})

	input := InteropMigrationInput{
		Prank: common.HexToAddress(l1ProxyAdminOwnerFlag),
		Opcm:  opcmAddr,
		MigrateInputV2: &MigrateInputV2{
			ChainSystemConfigs: chainSystemConfigs,
			DisputeGameConfigs: disputeGameConfigs,
			StartingAnchorRoot: Proposal{
				Root:             common.HexToHash(startingAnchorRootFlag),
				L2SequenceNumber: new(big.Int).SetUint64(cliCtx.Uint64(StartingAnchorL2SequenceNumberFlag.Name)),
			},
			StartingRespectedGameType: gameTypeZKDisputeGame,
		},
	}

	artifactsLocatorStr := cliCtx.String(deployer.ArtifactsLocatorFlag.Name)
	artifactsLocator := new(artifacts.Locator)
	if err := artifactsLocator.UnmarshalText([]byte(artifactsLocatorStr)); err != nil {
		return fmt.Errorf("failed to parse artifacts locator: %w", err)
	}

	l1RPC, err := rpc.Dial(l1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to dial RPC %s: %w", l1RPCUrl, err)
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

	l1Host, err := env.DefaultForkedScriptHost(ctx, bcaster, lgr, deployerAddr, artifactsFS, l1RPC)
	if err != nil {
		return fmt.Errorf("failed to create script host: %w", err)
	}

	output, err := SetInteropDisputeGames(l1Host, input)
	if err != nil {
		return fmt.Errorf("failed to run interop dispute game swap: %w", err)
	}

	enc := json.NewEncoder(cliCtx.App.Writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("failed to encode swap output: %w", err)
	}

	return nil
}
