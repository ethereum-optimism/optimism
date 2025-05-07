package bootstrap

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

// ConsolidatedOutput combines the outputs from Superchain and Implementations bootstrap
type ConsolidatedOutput struct {
	Superchain      opcm.DeploySuperchainOutput      `json:"superchain"`
	Implementations opcm.DeployImplementationsOutput `json:"implementations"`
}

type ConsolidatedConfig struct {
	L1RPCUrl                        string
	PrivateKey                      string
	Logger                          log.Logger
	ArtifactsLocator                *artifacts.Locator
	Outfile                         string
	MIPSVersion                     int
	WithdrawalDelaySeconds          uint64
	MinProposalSizeBytes            uint64
	ChallengePeriodSeconds          uint64
	ProofMaturityDelaySeconds       uint64
	DisputeGameFinalityDelaySeconds uint64
	SuperchainProxyAdminOwner       common.Address
	ProtocolVersionsOwner           common.Address
	Guardian                        common.Address
	Paused                          bool
	RequiredProtocolVersion         string
	RecommendedProtocolVersion      string
	UseInterop                      bool
	CacheDir                        string

	privateKeyECDSA *ecdsa.PrivateKey
}

func (c *ConsolidatedConfig) Parse() error {
	if c.L1RPCUrl == "" {
		return fmt.Errorf("L1RPCUrl must be specified")
	}

	if c.PrivateKey == "" {
		return fmt.Errorf("PrivateKey must be specified")
	}

	var err error
	c.privateKeyECDSA, err = crypto.HexToECDSA(strings.TrimPrefix(c.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	if c.ArtifactsLocator == nil {
		c.ArtifactsLocator = artifacts.DefaultL1ContractsLocator
	}

	if c.MIPSVersion == 0 {
		c.MIPSVersion = int(standard.MIPSVersion)
	}

	if c.SuperchainProxyAdminOwner == (common.Address{}) {
		return fmt.Errorf("SuperchainProxyAdminOwner must be specified")
	}

	if c.ProtocolVersionsOwner == (common.Address{}) {
		return fmt.Errorf("ProtocolVersionsOwner must be specified")
	}

	if c.Guardian == (common.Address{}) {
		return fmt.Errorf("Guardian must be specified")
	}

	return nil
}

var ConsolidatedFlags = []cli.Flag{
	deployer.L1RPCURLFlag,
	deployer.PrivateKeyFlag,
	OutfileFlag,
	deployer.ArtifactsLocatorFlag,
	MIPSVersionFlag,
	WithdrawalDelaySecondsFlag,
	MinProposalSizeBytesFlag,
	ChallengePeriodSecondsFlag,
	ProofMaturityDelaySecondsFlag,
	DisputeGameFinalityDelaySecondsFlag,
	SuperchainProxyAdminOwnerFlag,
	ProtocolVersionsOwnerFlag,
	GuardianFlag,
	PausedFlag,
	RequiredProtocolVersionFlag,
	RecommendedProtocolVersionFlag,
	UseInteropFlag,
}

func ConsolidatedCLI(ctx *cli.Context) error {
	ctx.Context = ctxinterrupt.WithCancelOnInterrupt(ctx.Context)

	logCfg := oplog.ReadCLIConfig(ctx)
	lgr := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
	oplog.SetGlobalLogHandler(lgr.Handler())

	outfile := ctx.String(OutfileFlagName)
	l1RpcUrl := ctx.String(deployer.L1RPCURLFlagName)
	privateKey := ctx.String(deployer.PrivateKeyFlagName)
	cacheDir := ctx.String(deployer.CacheDirFlagName)

	artifactsURLStr := ctx.String(deployer.ArtifactsLocatorFlagName)
	artifactsLocator := new(artifacts.Locator)
	if err := artifactsLocator.UnmarshalText([]byte(artifactsURLStr)); err != nil {
		return fmt.Errorf("failed to parse artifacts URL: %w", err)
	}

	mipsVersionInt := ctx.Int(MIPSVersionFlagName)

	withdrawalDelaySeconds := ctx.Uint64(WithdrawalDelaySecondsFlagName)
	minProposalSizeBytes := ctx.Uint64(MinProposalSizeBytesFlagName)
	challengePeriodSeconds := ctx.Uint64(ChallengePeriodSecondsFlagName)
	proofMaturityDelaySeconds := ctx.Uint64(ProofMaturityDelaySecondsFlagName)
	disputeGameFinalityDelaySeconds := ctx.Uint64(DisputeGameFinalityDelaySecondsFlagName)

	paused := ctx.Bool(PausedFlagName)
	requiredProtocolVersion := ctx.String(RequiredProtocolVersionFlagName)
	recommendedProtocolVersion := ctx.String(RecommendedProtocolVersionFlagName)
	useInterop := ctx.Bool("use-interop")

	superchainProxyAdminOwner := common.HexToAddress(ctx.String(SuperchainProxyAdminOwnerFlagName))
	protocolVersionsOwner := common.HexToAddress(ctx.String(ProtocolVersionsOwnerFlagName))
	guardian := common.HexToAddress(ctx.String(GuardianFlagName))

	cfg := ConsolidatedConfig{
		L1RPCUrl:                        l1RpcUrl,
		PrivateKey:                      privateKey,
		Logger:                          lgr,
		ArtifactsLocator:                artifactsLocator,
		Outfile:                         outfile,
		MIPSVersion:                     mipsVersionInt,
		WithdrawalDelaySeconds:          withdrawalDelaySeconds,
		MinProposalSizeBytes:            minProposalSizeBytes,
		ChallengePeriodSeconds:          challengePeriodSeconds,
		ProofMaturityDelaySeconds:       proofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: disputeGameFinalityDelaySeconds,
		SuperchainProxyAdminOwner:       superchainProxyAdminOwner,
		ProtocolVersionsOwner:           protocolVersionsOwner,
		Guardian:                        guardian,
		Paused:                          paused,
		RequiredProtocolVersion:         requiredProtocolVersion,
		RecommendedProtocolVersion:      recommendedProtocolVersion,
		UseInterop:                      useInterop,
		CacheDir:                        cacheDir,
	}

	if err := cfg.Parse(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	output, err := Consolidated(ctx.Context, cfg)
	if err != nil {
		return fmt.Errorf("error executing consolidated bootstrap: %w", err)
	}

	if err := jsonutil.WriteJSON(output, ioutil.ToStdOutOrFileOrNoop(outfile, 0o755)); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func Consolidated(ctx context.Context, cfg ConsolidatedConfig) (ConsolidatedOutput, error) {
	var output ConsolidatedOutput

	// First deploy the Superchain
	superchainCfg := SuperchainConfig{
		L1RPCUrl:                  cfg.L1RPCUrl,
		PrivateKey:                cfg.PrivateKey,
		Logger:                    cfg.Logger,
		ArtifactsLocator:          cfg.ArtifactsLocator,
		CacheDir:                  cfg.CacheDir,
		SuperchainProxyAdminOwner: cfg.SuperchainProxyAdminOwner,
		ProtocolVersionsOwner:     cfg.ProtocolVersionsOwner,
		Guardian:                  cfg.Guardian,
		Paused:                    cfg.Paused,
	}

	// Convert string protocol versions to params.ProtocolVersion
	if cfg.RequiredProtocolVersion != "" {
		if err := superchainCfg.RequiredProtocolVersion.UnmarshalText([]byte(cfg.RequiredProtocolVersion)); err != nil {
			return output, fmt.Errorf("failed to parse required protocol version: %w", err)
		}
	} else {
		superchainCfg.RequiredProtocolVersion = params.OPStackSupport
	}

	if cfg.RecommendedProtocolVersion != "" {
		if err := superchainCfg.RecommendedProtocolVersion.UnmarshalText([]byte(cfg.RecommendedProtocolVersion)); err != nil {
			return output, fmt.Errorf("failed to parse recommended protocol version: %w", err)
		}
	} else {
		superchainCfg.RecommendedProtocolVersion = params.OPStackSupport
	}

	// Parse the private key (required for both steps)
	var err error
	superchainCfg.privateKeyECDSA, err = crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		return output, fmt.Errorf("failed to parse private key: %w", err)
	}

	superchainOutput, err := Superchain(ctx, superchainCfg)
	if err != nil {
		return output, fmt.Errorf("failed to bootstrap superchain: %w", err)
	}
	output.Superchain = superchainOutput

	// Now deploy the implementations
	implsCfg := ImplementationsConfig{
		L1RPCUrl:                        cfg.L1RPCUrl,
		PrivateKey:                      cfg.PrivateKey,
		Logger:                          cfg.Logger,
		ArtifactsLocator:                cfg.ArtifactsLocator,
		MIPSVersion:                     cfg.MIPSVersion,
		WithdrawalDelaySeconds:          cfg.WithdrawalDelaySeconds,
		MinProposalSizeBytes:            cfg.MinProposalSizeBytes,
		ChallengePeriodSeconds:          cfg.ChallengePeriodSeconds,
		ProofMaturityDelaySeconds:       cfg.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: cfg.DisputeGameFinalityDelaySeconds,
		SuperchainConfigProxy:           superchainOutput.SuperchainConfigProxy,
		ProtocolVersionsProxy:           superchainOutput.ProtocolVersionsProxy,
		UpgradeController:               cfg.SuperchainProxyAdminOwner,
		UseInterop:                      cfg.UseInterop,
		CacheDir:                        cfg.CacheDir,
	}

	// Parse the private key
	implsCfg.privateKeyECDSA, err = crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		return output, fmt.Errorf("failed to parse private key: %w", err)
	}

	implsOutput, err := Implementations(ctx, implsCfg)
	if err != nil {
		return output, fmt.Errorf("failed to bootstrap implementations: %w", err)
	}
	output.Implementations = implsOutput

	return output, nil
}
