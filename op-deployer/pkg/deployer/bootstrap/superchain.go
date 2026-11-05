package bootstrap

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/verify"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
)

type SuperchainConfig struct {
	L1RPCUrl         string
	PrivateKey       string
	Logger           log.Logger
	ArtifactsLocator *artifacts.Locator
	CacheDir         string

	privateKeyECDSA *ecdsa.PrivateKey

	SuperchainProxyAdminOwner common.Address
	Guardian                  common.Address
	Paused                    bool
}

func (c *SuperchainConfig) Check() error {
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

	if c.SuperchainProxyAdminOwner == (common.Address{}) {
		return fmt.Errorf("superchain proxy admin owner must be specified")
	}

	if c.Guardian == (common.Address{}) {
		return fmt.Errorf("guardian must be specified")
	}

	return nil
}

func SuperchainCLI(cliCtx *cli.Context) error {
	logCfg := logcli.ReadCLIConfig(cliCtx)
	l := logcli.NewLogger(logcli.AppOut(cliCtx), logCfg)
	logcli.SetGlobalLogHandler(l.Handler())

	l1RPCUrl := cliCtx.String(deployer.L1RPCURLFlagName)
	privateKey := cliCtx.String(deployer.PrivateKeyFlagName)
	artifactsURLStr := cliCtx.String(deployer.ArtifactsLocatorFlagName)
	artifactsLocator := new(artifacts.Locator)
	if err := artifactsLocator.UnmarshalText([]byte(artifactsURLStr)); err != nil {
		return fmt.Errorf("failed to parse artifacts URL: %w", err)
	}

	superchainProxyAdminOwner := common.HexToAddress(cliCtx.String(SuperchainProxyAdminOwnerFlagName))
	guardian := common.HexToAddress(cliCtx.String(GuardianFlagName))
	paused := cliCtx.Bool(PausedFlagName)
	outfile := cliCtx.String(OutfileFlagName)
	cacheDir := cliCtx.String(deployer.CacheDirFlag.Name)
	cfg := SuperchainConfig{
		L1RPCUrl:                  l1RPCUrl,
		PrivateKey:                privateKey,
		Logger:                    l,
		ArtifactsLocator:          artifactsLocator,
		CacheDir:                  cacheDir,
		SuperchainProxyAdminOwner: superchainProxyAdminOwner,
		Guardian:                  guardian,
		Paused:                    paused,
	}

	ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)
	dso, err := Superchain(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to deploy superchain: %w", err)
	}

	if err := jsonutil.WriteJSON(dso, ioutil.ToStdOutOrFileOrNoop(outfile, 0o755)); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if cliCtx.Bool(deployer.NoVerifyFlag.Name) {
		l.Warn("Contract verification skipped", "reason", "--no-verify was set")
		return nil
	}

	verifyFile := outfile
	if err := func() error {
		if verifyFile == "" || verifyFile == "-" {
			tmpFile, err := os.CreateTemp("", "op-deployer-superchain-*.json")
			if err != nil {
				return fmt.Errorf("failed to create temp file for verification: %w", err)
			}
			tmpPath := tmpFile.Name()
			tmpFile.Close()
			defer os.Remove(tmpPath)
			verifyFile = tmpPath
			if err := jsonutil.WriteJSON(dso, ioutil.ToBasicFile(tmpPath, 0o644)); err != nil {
				return fmt.Errorf("failed to write temp file for verification: %w", err)
			}
		}

		chainID, err := deployer.ChainIDFromRPC(ctx, l1RPCUrl)
		if err != nil {
			return fmt.Errorf("failed to get chain ID: %w", err)
		}

		return verify.AutoVerify(
			ctx,
			l,
			l1RPCUrl,
			bigs.Uint64Strict(chainID),
			verifyFile,
			outfile,
			cfg.ArtifactsLocator,
			cliCtx.String(deployer.VerifierTypeFlagName),
			cliCtx.String(deployer.VerifierUrlFlagName),
			cliCtx.String(deployer.VerifierAPIKeyFlagName),
		)
	}(); err != nil {
		verify.LogAutoVerifyFailure(l, outfile, err)
	}
	return nil
}

func Superchain(ctx context.Context, cfg SuperchainConfig) (opcm.DeploySuperchainOutput, error) {
	var dso opcm.DeploySuperchainOutput

	if err := cfg.Check(); err != nil {
		return dso, fmt.Errorf("invalid config for Superchain: %w", err)
	}

	lgr := cfg.Logger
	cacheDir := cfg.CacheDir
	artifactsFS, err := artifacts.Download(ctx, cfg.ArtifactsLocator, ioutil.BarProgressor(), cacheDir)
	if err != nil {
		return dso, fmt.Errorf("failed to download artifacts: %w", err)
	}

	input := opcm.DeploySuperchainInput{
		SuperchainProxyAdminOwner: cfg.SuperchainProxyAdminOwner,
		Guardian:                  cfg.Guardian,
		Paused:                    cfg.Paused,
	}

	forgeClient, err := opcm.NewForgeClient(artifactsFS, cfg.CacheDir, []opcm.ForgeScript{opcm.DeploySuperchainForgeScript})
	if err != nil {
		return dso, fmt.Errorf("failed to create forge client: %w", err)
	}

	forgeEnv := &opcm.ForgeEnv{
		Client:     forgeClient,
		Context:    ctx,
		L1RPCUrl:   cfg.L1RPCUrl,
		PrivateKey: cfg.PrivateKey,
		Logger:     lgr,
	}
	dso, err = opcm.DeploySuperchain(forgeEnv, input)
	if err != nil {
		return dso, err
	}

	lgr.Info("deployed superchain configuration")

	return dso, nil
}
