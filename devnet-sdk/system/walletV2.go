package system

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	_ WalletV2 = (*walletV2)(nil)
)

type walletV2 struct {
	priv   *ecdsa.PrivateKey
	client *sources.EthClient
	ctx    context.Context
}

func NewWalletV2FromWalletAndChain(ctx context.Context, wallet Wallet, chain Chain) (WalletV2, error) {
	if len(chain.Nodes()) == 0 {
		return nil, fmt.Errorf("failed to init walletV2: chain has zero nodes")
	}
	client, err := chain.Nodes()[0].Client()
	if err != nil {
		return nil, err
	}
	return &walletV2{
		priv:   wallet.PrivateKey(),
		client: client,
		ctx:    ctx,
	}, nil
}

func NewWalletV2(ctx context.Context, rpcURL string, priv *ecdsa.PrivateKey, clCfg *sources.EthClientConfig, log log.Logger) (*walletV2, error) {
	if clCfg == nil {
		clCfg = sources.DefaultEthClientConfig(10)
	}
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	cl, err := sources.NewEthClient(client.NewBaseRPCClient(rpcClient), log, nil, clCfg)
	if err != nil {
		return nil, err
	}
	return &walletV2{
		client: cl,
		priv:   priv,
		ctx:    ctx,
	}, nil
}

func (w *walletV2) PrivateKey() *ecdsa.PrivateKey {
	return w.priv
}

func (w *walletV2) Client() *sources.EthClient {
	return w.client
}

func (w *walletV2) Ctx() context.Context {
	return w.ctx
}
