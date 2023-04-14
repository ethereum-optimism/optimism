package l1

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	cll1 "github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func NewFetchingL1(ctx context.Context, logger log.Logger, cfg *config.Config) (derive.L1Fetcher, error) {
	rpcClient, err := rpc.DialContext(ctx, cfg.L1URL)
	if err != nil {
		return nil, err
	}

	client := ethclient.NewClient(rpcClient)
	oracle := cll1.NewCachingOracle(NewFetchingL1Oracle(ctx, logger, client))
	return cll1.NewOracleL1Client(logger, oracle, cfg.L1Head), err
}
