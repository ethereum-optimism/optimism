package pipeline

import (
	"context"
	"fmt"
	"math/big"

	broadcaster2 "github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type BroadcasterFactory func(opts CallScriptBroadcastOpts) (broadcaster2.Broadcaster, error)

func KeyedBroadcaster(opts CallScriptBroadcastOpts) (broadcaster2.Broadcaster, error) {
	return broadcaster2.NewKeyedBroadcaster(broadcaster2.KeyedBroadcasterOpts{
		Logger:  opts.Logger,
		ChainID: opts.L1ChainID,
		Client:  opts.Client,
		Signer:  opts.Signer,
		From:    opts.Deployer,
	})
}

func DiscardBroadcaster(opts CallScriptBroadcastOpts) (broadcaster2.Broadcaster, error) {
	return broadcaster2.DiscardBroadcaster(), nil
}

type CallScriptBroadcastOpts struct {
	L1ChainID   *big.Int
	Logger      log.Logger
	ArtifactsFS foundry.StatDirFs
	Deployer    common.Address
	Signer      opcrypto.SignerFn
	Client      *ethclient.Client
	Handler     func(host *script.Host) error
	Broadcaster BroadcasterFactory
}

func CallScriptBroadcast(
	ctx context.Context,
	opts CallScriptBroadcastOpts,
) error {
	bcaster, err := opts.Broadcaster(opts)
	if err != nil {
		return fmt.Errorf("failed to create broadcaster: %w", err)
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
		script.WithCreate2Deployer(),
	)

	if err := h.EnableCheats(); err != nil {
		return fmt.Errorf("failed to enable cheats: %w", err)
	}

	nonce, err := opts.Client.NonceAt(ctx, opts.Deployer, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch nonce: %w", err)
	}
	h.SetNonce(opts.Deployer, nonce)

	err = opts.Handler(h)
	if err != nil {
		return fmt.Errorf("failed to run handler: %w", err)
	}

	if _, err := bcaster.Broadcast(ctx); err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	return nil
}
