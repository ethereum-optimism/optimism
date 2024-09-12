package opsm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type DeploySuperchainInput struct {
	ProxyAdminOwner            common.Address         `toml:"proxyAdminOwner"`
	ProtocolVersionsOwner      common.Address         `toml:"protocolVersionsOwner"`
	Guardian                   common.Address         `toml:"guardian"`
	Paused                     bool                   `toml:"paused"`
	RequiredProtocolVersion    params.ProtocolVersion `toml:"requiredProtocolVersion"`
	RecommendedProtocolVersion params.ProtocolVersion `toml:"recommendedProtocolVersion"`
}

func (dsi *DeploySuperchainInput) InputSet() bool {
	return true
}

type DeploySuperchainOutput struct {
	SuperchainProxyAdmin  common.Address `toml:"superchainProxyAdmin"`
	SuperchainConfigImpl  common.Address `toml:"superchainConfigImpl"`
	SuperchainConfigProxy common.Address `toml:"superchainConfigProxy"`
	ProtocolVersionsImpl  common.Address `toml:"protocolVersionsImpl"`
	ProtocolVersionsProxy common.Address `toml:"protocolVersionsProxy"`
}

func (output *DeploySuperchainOutput) CheckOutput() error {
	return nil
}

type DeploySuperchainScript struct {
	Run func(in common.Address, out common.Address) error
}

type DeploySuperchainOpts struct {
	ChainID     *big.Int
	ArtifactsFS foundry.StatDirFs
	Deployer    common.Address
	Signer      opcrypto.SignerFn
	Input       DeploySuperchainInput
	Client      *ethclient.Client
	Logger      log.Logger
}

func DeploySuperchainForge(ctx context.Context, opts DeploySuperchainOpts) (DeploySuperchainOutput, error) {
	var dso DeploySuperchainOutput

	bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  opts.Logger,
		ChainID: opts.ChainID,
		Client:  opts.Client,
		Signer:  opts.Signer,
		From:    opts.Deployer,
	})
	if err != nil {
		return dso, fmt.Errorf("failed to create broadcaster: %w", err)
	}

	scriptCtx := script.DefaultContext
	scriptCtx.Sender = opts.Deployer
	scriptCtx.Origin = opts.Deployer
	artifacts := &foundry.ArtifactsFS{FS: opts.ArtifactsFS}
	h := script.NewHost(
		opts.Logger,
		artifacts,
		nil,
		scriptCtx,
		script.WithBroadcastHook(bcaster.Hook),
		script.WithIsolatedBroadcasts(),
	)

	if err := h.EnableCheats(); err != nil {
		return dso, fmt.Errorf("failed to enable cheats: %w", err)
	}

	nonce, err := opts.Client.NonceAt(ctx, opts.Deployer, nil)
	if err != nil {
		return dso, fmt.Errorf("failed to get deployer nonce: %w", err)
	}

	inputAddr := h.NewScriptAddress()
	outputAddr := h.NewScriptAddress()

	cleanupInput, err := script.WithPrecompileAtAddress[*DeploySuperchainInput](h, inputAddr, &opts.Input)
	if err != nil {
		return dso, fmt.Errorf("failed to insert DeploySuperchainInput precompile: %w", err)
	}
	defer cleanupInput()

	cleanupOutput, err := script.WithPrecompileAtAddress[*DeploySuperchainOutput](
		h,
		outputAddr,
		&dso,
		script.WithFieldSetter[*DeploySuperchainOutput],
	)
	if err != nil {
		return dso, fmt.Errorf("failed to insert DeploySuperchainOutput precompile: %w", err)
	}
	defer cleanupOutput()

	deployScript, cleanupDeploy, err := script.WithScript[DeploySuperchainScript](h, "DeploySuperchain.s.sol", "DeploySuperchain")
	if err != nil {
		return dso, fmt.Errorf("failed to load DeploySuperchain script: %w", err)
	}
	defer cleanupDeploy()

	h.SetNonce(opts.Deployer, nonce)

	opts.Logger.Info("deployer nonce", "nonce", nonce)

	if err := deployScript.Run(inputAddr, outputAddr); err != nil {
		return dso, fmt.Errorf("failed to run DeploySuperchain script: %w", err)
	}

	if _, err := bcaster.Broadcast(ctx); err != nil {
		return dso, fmt.Errorf("failed to broadcast transactions: %w", err)
	}

	return dso, nil
}
