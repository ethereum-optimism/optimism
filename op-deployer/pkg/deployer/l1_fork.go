package deployer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// l1Fork simulates deployments on an ephemeral anvil fork of the L1 head. forge sends from the
// unlocked deployer, so no key is needed, and each deployment stays on the fork for the next.
type l1Fork struct {
	fork   *forge.Fork
	client *ethclient.Client
	// forge runs the scripts on the fork.
	forge *opcm.ForgeEnv
	// block is the last block whose transactions are already returned.
	block uint64
}

func startL1Fork(
	ctx context.Context,
	l1Client *ethclient.Client,
	l1RPCURL string,
	deployer common.Address,
	forgeClient *forge.Client,
	cacheDir string,
) (*l1Fork, error) {
	head, err := l1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read the L1 head: %w", err)
	}
	var binOpts []forge.StandardBinOpt
	if cacheDir != "" {
		binOpts = append(binOpts, forge.WithCachePather(func() (string, error) { return cacheDir, nil }))
	}
	anvil, err := forge.NewAnvilBinary(binOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create anvil binary: %w", err)
	}
	headNumber := bigs.Uint64Strict(head.Number)
	fork, err := forge.StartFork(ctx, anvil, l1RPCURL, headNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fork L1: %w", err)
	}
	client, err := ethclient.DialContext(ctx, fork.RPCURL())
	if err != nil {
		_ = fork.Stop()
		return nil, fmt.Errorf("failed to connect to the L1 fork: %w", err)
	}
	return &l1Fork{
		fork:   fork,
		client: client,
		forge: &opcm.ForgeEnv{
			Client:         forgeClient,
			Context:        ctx,
			L1RPCUrl:       fork.RPCURL(),
			UnlockedSender: deployer,
		},
		block: headNumber,
	}, nil
}

// transactions returns the transactions of the fork since the previous call, oldest first.
func (f *l1Fork) transactions(ctx context.Context) ([]broadcaster.Broadcast, error) {
	head, err := f.client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the fork head: %w", err)
	}
	var bcasts []broadcaster.Broadcast
	for number := f.block + 1; number <= head; number++ {
		block, err := f.client.BlockByNumber(ctx, new(big.Int).SetUint64(number))
		if err != nil {
			return nil, fmt.Errorf("failed to read fork block %d: %w", number, err)
		}
		for i, tx := range block.Transactions() {
			// anvil signs impersonated transactions with a placeholder, so the sender comes from the RPC.
			from, err := f.client.TransactionSender(ctx, tx, block.Hash(), uint(i))
			if err != nil {
				return nil, fmt.Errorf("failed to read the sender of fork transaction %s: %w", tx.Hash(), err)
			}
			bcast := broadcaster.Broadcast{
				From:  from,
				Input: tx.Data(),
				Value: (*hexutil.U256)(uint256.MustFromBig(tx.Value())),
				Type:  broadcaster.BroadcastCall,
				Nonce: tx.Nonce(),
			}
			if tx.To() == nil {
				bcast.Type = broadcaster.BroadcastCreate
			} else {
				bcast.To = *tx.To()
			}
			bcasts = append(bcasts, bcast)
		}
	}
	f.block = head
	return bcasts, nil
}

func (f *l1Fork) stop() error {
	f.client.Close()
	return f.fork.Stop()
}
