package bootstrap

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"

	mipsVersion "github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/cliutil"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

type ImplementationsConfig struct {
	L1RPCUrl                        string             `cli:"l1-rpc-url"`
	PrivateKey                      string             `cli:"private-key"`
	ArtifactsLocator                *artifacts.Locator `cli:"artifacts-locator"`
	MIPSVersion                     int                `cli:"mips-version"`
	WithdrawalDelaySeconds          uint64             `cli:"withdrawal-delay-seconds"`
	MinProposalSizeBytes            uint64             `cli:"min-proposal-size-bytes"`
	ChallengePeriodSeconds          uint64             `cli:"challenge-period-seconds"`
	ProofMaturityDelaySeconds       uint64             `cli:"proof-maturity-delay-seconds"`
	DisputeGameFinalityDelaySeconds uint64             `cli:"dispute-game-finality-delay-seconds"`
	SuperchainConfigProxy           common.Address     `cli:"superchain-config-proxy"`
	ProtocolVersionsProxy           common.Address     `cli:"protocol-versions-proxy"`
	UpgradeController               common.Address     `cli:"upgrade-controller"`
	SuperchainProxyAdmin            common.Address     `cli:"superchain-proxy-admin"`
	Challenger                      common.Address     `cli:"challenger"`
	CacheDir                        string             `cli:"cache-dir"`

	Logger log.Logger

	privateKeyECDSA *ecdsa.PrivateKey
}

func (c *ImplementationsConfig) Check() error {
	if c.L1RPCUrl == "" {
		return errors.New("l1RPCUrl must be specified")
	}
	if c.PrivateKey == "" {
		return errors.New("private key must be specified")
	}

	privECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(c.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	c.privateKeyECDSA = privECDSA

	if c.Logger == nil {
		return errors.New("logger must be specified")
	}
	if c.ArtifactsLocator == nil {
		return errors.New("artifacts locator must be specified")
	}
	if !mipsVersion.IsSupported(c.MIPSVersion) {
		return errors.New("MIPS version is not supported")
	}
	if c.WithdrawalDelaySeconds == 0 {
		return errors.New("withdrawal delay in seconds must be specified")
	}
	if c.MinProposalSizeBytes == 0 {
		return errors.New("preimage oracle minimum proposal size in bytes must be specified")
	}
	if c.ChallengePeriodSeconds == 0 {
		return errors.New("preimage oracle challenge period in seconds must be specified")
	}
	if c.ProofMaturityDelaySeconds == 0 {
		return errors.New("proof maturity delay in seconds must be specified")
	}
	if c.DisputeGameFinalityDelaySeconds == 0 {
		return errors.New("dispute game finality delay in seconds must be specified")
	}
	if c.SuperchainConfigProxy == (common.Address{}) {
		return errors.New("superchain config proxy must be specified")
	}
	if c.ProtocolVersionsProxy == (common.Address{}) {
		return errors.New("protocol versions proxy must be specified")
	}
	if c.UpgradeController == (common.Address{}) {
		return errors.New("upgrade controller must be specified")
	}
	if c.SuperchainProxyAdmin == (common.Address{}) {
		return errors.New("superchain proxy admin must be specified")
	}
	if c.Challenger == (common.Address{}) {
		return errors.New("challenger must be specified")
	}
	return nil
}

func ImplementationsCLI(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	var cfg ImplementationsConfig
	if err := cliutil.PopulateStruct(&cfg, cliCtx); err != nil {
		return fmt.Errorf("failed to populate config: %w", err)
	}
	cfg.Logger = l

	artifactsURLStr := cliCtx.String(deployer.ArtifactsLocatorFlagName)
	artifactsLocator := new(artifacts.Locator)
	if err := artifactsLocator.UnmarshalText([]byte(artifactsURLStr)); err != nil {
		return fmt.Errorf("failed to parse artifacts URL: %w", err)
	}
	cfg.ArtifactsLocator = artifactsLocator

	ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)
	outfile := cliCtx.String(OutfileFlagName)
	dio, err := Implementations(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to deploy implementations: %w", err)
	}
	if err := jsonutil.WriteJSON(dio, ioutil.ToStdOutOrFileOrNoop(outfile, 0o755)); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

func Implementations(ctx context.Context, cfg ImplementationsConfig) (opcm.DeployImplementationsOutput, error) {
	var dio opcm.DeployImplementationsOutput
	if err := cfg.Check(); err != nil {
		return dio, fmt.Errorf("invalid config for Implementations: %w", err)
	}

	lgr := cfg.Logger

	artifactsFS, err := artifacts.Download(ctx, cfg.ArtifactsLocator, ioutil.BarProgressor(), cfg.CacheDir)
	if err != nil {
		return dio, fmt.Errorf("failed to download artifacts: %w", err)
	}

	l1Client, err := ethclient.Dial(cfg.L1RPCUrl)
	if err != nil {
		return dio, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}

	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return dio, fmt.Errorf("failed to get chain ID: %w", err)
	}

	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(cfg.privateKeyECDSA, chainID))
	chainDeployer := crypto.PubkeyToAddress(cfg.privateKeyECDSA.PublicKey)

	bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  lgr,
		ChainID: chainID,
		Client:  l1Client,
		Signer:  signer,
		From:    chainDeployer,
	})
	if err != nil {
		return dio, fmt.Errorf("failed to create broadcaster: %w", err)
	}

	l1RPC, err := rpc.Dial(cfg.L1RPCUrl)
	if err != nil {
		return dio, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}

	l1Host, err := env.DefaultForkedScriptHost(
		ctx,
		bcaster,
		lgr,
		chainDeployer,
		artifactsFS,
		l1RPC,
	)
	if err != nil {
		return dio, fmt.Errorf("failed to create script host: %w", err)
	}

	opcmScripts, err := opcm.NewScripts(l1Host)
	if err != nil {
		return dio, fmt.Errorf("failed to load OPCM scripts: %w", err)
	}

	if dio, err = opcmScripts.DeployImplementations.Run(
		opcm.DeployImplementationsInput{
			WithdrawalDelaySeconds:          new(big.Int).SetUint64(cfg.WithdrawalDelaySeconds),
			MinProposalSizeBytes:            new(big.Int).SetUint64(cfg.MinProposalSizeBytes),
			ChallengePeriodSeconds:          new(big.Int).SetUint64(cfg.ChallengePeriodSeconds),
			ProofMaturityDelaySeconds:       new(big.Int).SetUint64(cfg.ProofMaturityDelaySeconds),
			DisputeGameFinalityDelaySeconds: new(big.Int).SetUint64(cfg.DisputeGameFinalityDelaySeconds),
			MipsVersion:                     new(big.Int).SetUint64(uint64(cfg.MIPSVersion)),
			SuperchainConfigProxy:           cfg.SuperchainConfigProxy,
			ProtocolVersionsProxy:           cfg.ProtocolVersionsProxy,
			SuperchainProxyAdmin:            cfg.SuperchainProxyAdmin,
			UpgradeController:               cfg.UpgradeController,
			Challenger:                      cfg.Challenger,
		},
	); err != nil {
		return dio, fmt.Errorf("error deploying implementations: %w", err)
	}

	if _, err := bcaster.Broadcast(ctx); err != nil {
		return dio, fmt.Errorf("failed to broadcast: %w", err)
	}

	lgr.Info("deployed implementations")
	return dio, nil
}
