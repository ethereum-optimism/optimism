package pipeline

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
)

type BroadcastCfg struct {
	Client *ethclient.Client
	Signer opcrypto.SignerFn
}

func NewL1Broadcaster(cfg *BroadcastCfg, lgr log.Logger, chainID *big.Int, deployer common.Address) (broadcaster.Broadcaster, error) {
	if cfg == nil {
		return broadcaster.DiscardBroadcaster(), nil
	}

	return broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  lgr,
		ChainID: chainID,
		Client:  cfg.Client,
		Signer:  cfg.Signer,
		From:    deployer,
	})
}

type CallScriptBroadcastOpts struct {
	Logger      log.Logger
	ArtifactsFS foundry.StatDirFs
	Deployer    common.Address
	Handler     func(host *script.Host) error
	Broadcaster broadcaster.Broadcaster
}

func CallScriptBroadcast(
	ctx context.Context,
	opts CallScriptBroadcastOpts,
) error {

	scriptCtx := script.DefaultContext
	scriptCtx.Sender = opts.Deployer
	scriptCtx.Origin = opts.Deployer
	artifacts := &foundry.ArtifactsFS{FS: opts.ArtifactsFS}
	h := script.NewHost(
		opts.Logger,
		artifacts,
		nil,
		scriptCtx,
		script.WithBroadcastHook(opts.Broadcaster.Hook),
		script.WithIsolatedBroadcasts(),
		script.WithCreate2Deployer(),
	)

	if err := h.EnableCheats(); err != nil {
		return fmt.Errorf("failed to enable cheats: %w", err)
	}

	if err := opts.Broadcaster.PrepareHost(ctx, h); err != nil {
		return fmt.Errorf("failed to prepare host: %w", err)
	}

	err := opts.Handler(h)
	if err != nil {
		return fmt.Errorf("failed to run handler: %w", err)
	}

	if _, err := opts.Broadcaster.Broadcast(ctx); err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	return nil
}
