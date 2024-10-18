package bootstrap

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"

	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

type OPCMConfig struct {
	L1RPCUrl         string
	PrivateKey       string
	Logger           log.Logger
	ArtifactsLocator *opcm.ArtifactsLocator

	privateKeyECDSA *ecdsa.PrivateKey
}

func (c *OPCMConfig) Check() error {
	if c.L1RPCUrl == "" {
		return fmt.Errorf("l1RPCUrl must be specified")
	}

	if c.PrivateKey == "" {
		return fmt.Errorf("private key must be specified")
	}

	privECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(c.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	c.privateKeyECDSA = privECDSA

	if c.Logger == nil {
		return fmt.Errorf("logger must be specified")
	}

	if c.ArtifactsLocator == nil {
		return fmt.Errorf("artifacts locator must be specified")
	}

	return nil
}

func OPCMCLI(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	l1RPCUrl := cliCtx.String(deployer.L1RPCURLFlagName)
	privateKey := cliCtx.String(deployer.PrivateKeyFlagName)
	artifactsURLStr := cliCtx.String(ArtifactsLocatorFlagName)
	artifactsLocator := new(opcm.ArtifactsLocator)
	if err := artifactsLocator.UnmarshalText([]byte(artifactsURLStr)); err != nil {
		return fmt.Errorf("failed to parse artifacts URL: %w", err)
	}

	ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)

	return OPCM(ctx, OPCMConfig{
		L1RPCUrl:         l1RPCUrl,
		PrivateKey:       privateKey,
		Logger:           l,
		ArtifactsLocator: artifactsLocator,
	})
}

func OPCM(ctx context.Context, cfg OPCMConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for OPCM: %w", err)
	}

	lgr := cfg.Logger
	progressor := func(curr, total int64) {
		lgr.Info("artifacts download progress", "current", curr, "total", total)
	}

	artifactsFS, cleanup, err := pipeline.DownloadArtifacts(ctx, cfg.ArtifactsLocator, progressor)
	if err != nil {
		return fmt.Errorf("failed to download artifacts: %w", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			lgr.Warn("failed to clean up artifacts", "err", err)
		}
	}()

	l1Client, err := ethclient.Dial(cfg.L1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}

	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}
	chainIDU64 := chainID.Uint64()

	superCfg, err := opcm.SuperchainFor(chainIDU64)
	if err != nil {
		return fmt.Errorf("error getting superchain config: %w", err)
	}
	standardVersionsTOML, err := opcm.StandardL1VersionsDataFor(chainIDU64)
	if err != nil {
		return fmt.Errorf("error getting standard versions TOML: %w", err)
	}
	opcmProxyOwnerAddr, err := opcm.ManagerOwnerAddrFor(chainIDU64)
	if err != nil {
		return fmt.Errorf("error getting superchain proxy admin: %w", err)
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
		return fmt.Errorf("failed to create broadcaster: %w", err)
	}

	nonce, err := l1Client.NonceAt(ctx, chainDeployer, nil)
	if err != nil {
		return fmt.Errorf("failed to get starting nonce: %w", err)
	}

	host, err := pipeline.DefaultScriptHost(
		bcaster,
		lgr,
		chainDeployer,
		artifactsFS,
		nonce,
	)
	if err != nil {
		return fmt.Errorf("failed to create script host: %w", err)
	}

	var release string
	if cfg.ArtifactsLocator.IsTag() {
		release = cfg.ArtifactsLocator.Tag
	} else {
		release = "dev"
	}

	lgr.Info("deploying OPCM", "release", release)

	// We need to etch the Superchain addresses so that they have nonzero code
	// and the checks in the OPCM constructor pass.
	superchainConfigAddr := common.Address(*superCfg.Config.SuperchainConfigAddr)
	protocolVersionsAddr := common.Address(*superCfg.Config.ProtocolVersionsAddr)
	addresses := []common.Address{
		superchainConfigAddr,
		protocolVersionsAddr,
	}
	for _, addr := range addresses {
		host.ImportAccount(addr, types.Account{
			Code: []byte{0x00},
		})
	}

	var salt common.Hash
	_, err = rand.Read(salt[:])
	if err != nil {
		return fmt.Errorf("failed to generate CREATE2 salt: %w", err)
	}

	dio, err := opcm.DeployImplementations(
		host,
		opcm.DeployImplementationsInput{
			Salt:                            salt,
			WithdrawalDelaySeconds:          big.NewInt(604800),
			MinProposalSizeBytes:            big.NewInt(126000),
			ChallengePeriodSeconds:          big.NewInt(86400),
			ProofMaturityDelaySeconds:       big.NewInt(604800),
			DisputeGameFinalityDelaySeconds: big.NewInt(302400),
			Release:                         release,
			SuperchainConfigProxy:           superchainConfigAddr,
			ProtocolVersionsProxy:           protocolVersionsAddr,
			OpcmProxyOwner:                  opcmProxyOwnerAddr,
			StandardVersionsToml:            standardVersionsTOML,
			UseInterop:                      false,
		},
	)
	if err != nil {
		return fmt.Errorf("error deploying implementations: %w", err)
	}

	if _, err := bcaster.Broadcast(ctx); err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	lgr.Info("deployed implementations")

	if err := jsonutil.WriteJSON(dio, ioutil.ToStdOut()); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}
