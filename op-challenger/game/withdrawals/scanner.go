package withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

type L1Client interface {
	FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]ethTypes.Log, error)
}

// proofs finds the withdrawal proofs submitted between from and to inclusive. A proof is only
// invalidated by a game if it was submitted after the game was created, so callers scan from the
// game's L1 head.
//
// The range is filtered by block number rather than pinned to the head's hash because
// FilterQuery.BlockHash only matches a single block.
func (d *Deleter) proofs(ctx context.Context, from uint64, to uint64) ([]contracts.WithdrawalProof, error) {
	if from > to {
		// L1 reorged below the cursor, so rescan from the new head.
		from = to
	}
	logs, err := d.l1.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(to),
		Addresses: []common.Address{d.portal.Addr()},
		Topics:    [][]common.Hash{{d.portal.WithdrawalProvenExtension1Topic()}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch withdrawal proof logs from %v to %v: %w", from, to, err)
	}
	proofs := make([]contracts.WithdrawalProof, 0, len(logs))
	seen := make(map[contracts.WithdrawalProof]bool, len(logs))
	for i := range logs {
		proof, err := d.portal.DecodeWithdrawalProvenExtension1(&logs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to decode withdrawal proof log: %w", err)
		}
		if seen[proof] {
			continue
		}
		seen[proof] = true
		proofs = append(proofs, proof)
	}
	return proofs, nil
}
