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

type InteropMigrationInput struct {
	Prank common.Address `json:"prank"`
	Opcm  common.Address `json:"opcm"`

	UsePermissionlessGame          bool           `json:"usePermissionlessGame"`
	StartingAnchorRoot             common.Hash    `json:"startingAnchorRoot"`
	StartingAnchorL2SequenceNumber *big.Int       `json:"startingAnchorL2SequenceNumber"`
	Proposer                       common.Address `json:"proposer"`
	Challenger                     common.Address `json:"challenger"`
	MaxGameDepth                   uint64         `json:"maxGameDepth"`
	SplitDepth                     uint64         `json:"splitDepth"`
	InitBond                       *big.Int       `json:"initBond"`
	ClockExtension                 uint64         `json:"clockExtension"`
	MaxClockDuration               uint64         `json:"maxClockDuration"`

	EncodedChainConfigs []OPChainConfig `evm:"-" json:"chainConfigs"`
}

func (u *InteropMigrationInput) OpChainConfigs() ([]byte, error) {
	data, err := opChainConfigEncoder.EncodeArgs(u.EncodedChainConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to encode chain configs: %w", err)
	}
	return data[4:], nil
}

type OPChainConfig struct {
	SystemConfigProxy common.Address `json:"systemConfigProxy"`
	ProxyAdmin        common.Address `json:"proxyAdmin"`
	AbsolutePrestate  common.Hash    `json:"absolutePrestate"`
}

type InteropMigrationOutput struct {
	DisputeGameFactory common.Address `json:"disputeGameFactory"`
}

func (output *InteropMigrationOutput) CheckOutput(input common.Address) error {
	return nil
}

var opChainConfigEncoder = w3.MustNewFunc("dummy((address systemConfigProxy,address proxyAdmin,bytes32 absolutePrestate)[])", "")

type InteropMigration struct {
	Run func(input common.Address)
}

func Migrate(host *script.Host, input InteropMigrationInput) (InteropMigrationOutput, error) {
	return opcm.RunScriptSingle[InteropMigrationInput, InteropMigrationOutput](host, input, "InteropMigration.s.sol", "InteropMigration")
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

	input := InteropMigrationInput{
		Prank:                          common.Address{}, // The current CLI does not support prank address, so we set it to zero.
		Opcm:                           common.HexToAddress(cliCtx.String(OPCMImplFlag.Name)),
		UsePermissionlessGame:          cliCtx.Bool(PermissionlessFlag.Name),
		StartingAnchorRoot:             common.HexToHash(cliCtx.String(StartingAnchorRootFlag.Name)),
		StartingAnchorL2SequenceNumber: new(big.Int).SetUint64(cliCtx.Uint64(StartingAnchorL2SequenceNumberFlag.Name)),
		Proposer:                       common.HexToAddress(cliCtx.String(ProposerFlag.Name)),
		Challenger:                     common.HexToAddress(cliCtx.String(ChallengerFlag.Name)),
		MaxGameDepth:                   cliCtx.Uint64(DisputeMaxGameDepthFlag.Name),
		SplitDepth:                     cliCtx.Uint64(DisputeSplitDepthFlag.Name),
		InitBond:                       big.NewInt(int64(cliCtx.Uint64(InitialBondFlag.Name))),
		ClockExtension:                 cliCtx.Uint64(DisputeClockExtensionFlag.Name),
		MaxClockDuration:               cliCtx.Uint64(DisputeMaxClockDurationFlag.Name),
		// At the moment we only support a single chain config
		EncodedChainConfigs: []OPChainConfig{
			{
				SystemConfigProxy: common.HexToAddress(cliCtx.String(SystemConfigProxyFlag.Name)),
				ProxyAdmin:        common.HexToAddress(cliCtx.String(OPChainProxyAdminFlag.Name)),
				AbsolutePrestate:  common.HexToHash(cliCtx.String(DisputeAbsolutePrestateFlag.Name)),
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

	l1RPC, err := rpc.Dial(l1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to dial RPC %s: %w", l1RPCUrl, err)
	}

	l1Client := ethclient.NewClient(l1RPC)
	l1ChainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(privateKeyECDSA, l1ChainID))
	deployer := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
	bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  lgr,
		ChainID: l1ChainID,
		Client:  l1Client,
		Signer:  signer,
		From:    deployer,
	})
	if err != nil {
		return fmt.Errorf("failed to create broadcaster: %w", err)
	}

	l1Host, err := env.DefaultForkedScriptHost(
		ctx,
		bcaster,
		lgr,
		deployer,
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
