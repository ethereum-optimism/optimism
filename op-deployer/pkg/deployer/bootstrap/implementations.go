package bootstrap

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	mipsVersion "github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/verify"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
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
	DevFeatureBitmap                common.Hash        `cli:"dev-feature-bitmap"`
	OutputRootBootstrap             bool               `cli:"output-root-bootstrap"`
	SP1Verifier                     common.Address     `cli:"sp1-verifier-address"`
	FaultGameMaxGameDepth           uint64             `cli:"dispute-max-game-depth"`
	FaultGameSplitDepth             uint64             `cli:"dispute-split-depth"`
	FaultGameClockExtension         uint64             `cli:"dispute-clock-extension"`
	FaultGameMaxClockDuration       uint64             `cli:"dispute-max-clock-duration"`
	SuperchainConfigProxy           common.Address     `cli:"superchain-config-proxy"`
	L1ProxyAdminOwner               common.Address     `cli:"l1-proxy-admin-owner"`
	SuperchainProxyAdmin            common.Address     `cli:"superchain-proxy-admin"`
	Challenger                      common.Address     `cli:"challenger"`
	CacheDir                        string             `cli:"cache-dir"`
	UseForge                        bool               `cli:"use-forge"`

	Logger log.Logger

	privateKeyECDSA *ecdsa.PrivateKey
}

const (
	// Source: succinctlabs/sp1-contracts@2ac5ecbbe473421a963d67e55f182e9a36576f7c,
	// contracts/deployments/1.json, V6_1_0_SP1_VERIFIER_PLONK.
	mainnetSP1VerifierV610 = "0xc3c6dDDAc8829b233Dc6536Ec024775a57b0AF2A"
	// Succinct deploys the same verifier bytecode and address deterministically on both networks.
	// Source: succinctlabs/sp1-contracts@2ac5ecbbe473421a963d67e55f182e9a36576f7c,
	// contracts/deployments/11155111.json, V6_1_0_SP1_VERIFIER_PLONK.
	sepoliaSP1VerifierV610 = "0xc3c6dDDAc8829b233Dc6536Ec024775a57b0AF2A"
)

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
	if c.OutputRootBootstrap {
		if devfeatures.EnableDevFeature(c.DevFeatureBitmap, devfeatures.SuperRootGamesMigrationFlag) == c.DevFeatureBitmap {
			return errors.New("output root bootstrap conflicts with an explicit super root games migration feature")
		}
		c.DevFeatureBitmap = devfeatures.EnableDevFeature(c.DevFeatureBitmap, devfeatures.OutputRootGamesFlag)
	}
	if c.FaultGameMaxGameDepth == 0 {
		return errors.New("fault game max game depth must be specified")
	}
	if c.FaultGameSplitDepth == 0 {
		return errors.New("fault game split depth must be specified")
	}
	if c.FaultGameClockExtension == 0 {
		return errors.New("fault game clock extension must be specified")
	}
	if c.FaultGameMaxClockDuration == 0 {
		return errors.New("fault game max clock duration must be specified")
	}
	if c.SuperchainConfigProxy == (common.Address{}) {
		return errors.New("superchain config proxy must be specified")
	}
	if c.L1ProxyAdminOwner == (common.Address{}) {
		return errors.New("l1 proxy admin owner must be specified")
	}
	if c.SuperchainProxyAdmin == (common.Address{}) {
		return errors.New("superchain proxy admin must be specified")
	}
	if c.Challenger == (common.Address{}) {
		return errors.New("challenger must be specified")
	}
	zkEnabled := devfeatures.IsDevFeatureEnabled(c.DevFeatureBitmap, devfeatures.ZKDisputeGameFlag)
	if !zkEnabled && c.SP1Verifier != (common.Address{}) {
		return errors.New("sp1 verifier must not be specified when ZK dispute games are disabled")
	}
	return nil
}

func (c *ImplementationsConfig) resolveSP1Verifier(chainID *big.Int) error {
	if !devfeatures.IsDevFeatureEnabled(c.DevFeatureBitmap, devfeatures.ZKDisputeGameFlag) ||
		c.SP1Verifier != (common.Address{}) {
		return nil
	}

	if !chainID.IsUint64() {
		return fmt.Errorf(
			"no default SP1 verifier for L1 chain ID %s; specify --%s",
			chainID,
			SP1VerifierAddressFlagName,
		)
	}
	chainIDUint64 := bigs.Uint64Strict(chainID)
	switch chainIDUint64 {
	case 1:
		c.SP1Verifier = common.HexToAddress(mainnetSP1VerifierV610)
	case 11155111:
		c.SP1Verifier = common.HexToAddress(sepoliaSP1VerifierV610)
	default:
		return fmt.Errorf(
			"no default SP1 verifier for L1 chain ID %d; specify --%s",
			chainIDUint64,
			SP1VerifierAddressFlagName,
		)
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
	if cliCtx.IsSet(SP1VerifierAddressFlagName) && cfg.SP1Verifier == (common.Address{}) {
		return fmt.Errorf("--%s must not be zero when explicitly set", SP1VerifierAddressFlagName)
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

	if cliCtx.Bool(deployer.NoVerifyFlag.Name) {
		l.Warn("Contract verification skipped", "reason", "--no-verify was set")
		return nil
	}

	verifyFile := outfile
	if err := func() error {
		if verifyFile == "" || verifyFile == "-" {
			tmpFile, err := os.CreateTemp("", "op-deployer-implementations-*.json")
			if err != nil {
				return fmt.Errorf("failed to create temp file for verification: %w", err)
			}
			tmpPath := tmpFile.Name()
			tmpFile.Close()
			defer os.Remove(tmpPath)
			verifyFile = tmpPath
			if err := jsonutil.WriteJSON(dio, ioutil.ToBasicFile(tmpPath, 0o644)); err != nil {
				return fmt.Errorf("failed to write temp file for verification: %w", err)
			}
		}

		l1RPCUrl := cliCtx.String(deployer.L1RPCURLFlagName)
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

func Implementations(ctx context.Context, cfg ImplementationsConfig) (opcm.DeployImplementationsOutput, error) {
	var dio opcm.DeployImplementationsOutput
	if err := cfg.Check(); err != nil {
		return dio, fmt.Errorf("invalid config for Implementations: %w", err)
	}
	zkEnabled := devfeatures.IsDevFeatureEnabled(cfg.DevFeatureBitmap, devfeatures.ZKDisputeGameFlag)
	if zkEnabled && cfg.SP1Verifier == (common.Address{}) {
		chainID, err := deployer.ChainIDFromRPC(ctx, cfg.L1RPCUrl)
		if err != nil {
			return dio, fmt.Errorf("failed to select default SP1 verifier: %w", err)
		}
		if err := cfg.resolveSP1Verifier(chainID); err != nil {
			return dio, err
		}
	}

	lgr := cfg.Logger
	if zkEnabled {
		lgr.Info("using SP1 verifier", "address", cfg.SP1Verifier)
	}

	artifactsFS, err := artifacts.Download(ctx, cfg.ArtifactsLocator, ioutil.BarProgressor(), cfg.CacheDir)
	if err != nil {
		return dio, fmt.Errorf("failed to download artifacts: %w", err)
	}

	input := opcm.DeployImplementationsInput{
		WithdrawalDelaySeconds:          new(big.Int).SetUint64(cfg.WithdrawalDelaySeconds),
		MinProposalSizeBytes:            new(big.Int).SetUint64(cfg.MinProposalSizeBytes),
		ChallengePeriodSeconds:          new(big.Int).SetUint64(cfg.ChallengePeriodSeconds),
		ProofMaturityDelaySeconds:       new(big.Int).SetUint64(cfg.ProofMaturityDelaySeconds),
		DisputeGameFinalityDelaySeconds: new(big.Int).SetUint64(cfg.DisputeGameFinalityDelaySeconds),
		MipsVersion:                     new(big.Int).SetUint64(uint64(cfg.MIPSVersion)),
		DevFeatureBitmap:                cfg.DevFeatureBitmap,
		FaultGameV2MaxGameDepth:         new(big.Int).SetUint64(cfg.FaultGameMaxGameDepth),
		FaultGameV2SplitDepth:           new(big.Int).SetUint64(cfg.FaultGameSplitDepth),
		FaultGameV2ClockExtension:       new(big.Int).SetUint64(cfg.FaultGameClockExtension),
		FaultGameV2MaxClockDuration:     new(big.Int).SetUint64(cfg.FaultGameMaxClockDuration),
		SuperchainConfigProxy:           cfg.SuperchainConfigProxy,
		SuperchainProxyAdmin:            cfg.SuperchainProxyAdmin,
		L1ProxyAdminOwner:               cfg.L1ProxyAdminOwner,
		Challenger:                      cfg.Challenger,
		SP1Verifier:                     cfg.SP1Verifier,
	}

	if cfg.UseForge {
		lgr.Info("using Forge for DeployImplementations")
		forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", artifactsFS))
		if err != nil {
			return dio, fmt.Errorf("failed to create forge client: %w", err)
		}

		forgeEnv := &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    ctx,
			L1RPCUrl:   cfg.L1RPCUrl,
			PrivateKey: cfg.PrivateKey,
		}
		dio, err = opcm.DeployImplementationsViaForge(forgeEnv, input)
		if err != nil {
			return dio, err
		}
	} else {
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

		if dio, err = opcmScripts.DeployImplementations.Run(input); err != nil {
			return dio, fmt.Errorf("error deploying implementations: %w", err)
		}

		if _, err := bcaster.Broadcast(ctx); err != nil {
			return dio, fmt.Errorf("failed to broadcast: %w", err)
		}
	}

	lgr.Info("deployed implementations")
	return dio, nil
}
