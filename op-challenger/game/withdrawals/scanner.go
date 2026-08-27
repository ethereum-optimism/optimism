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
	HeaderByHash(ctx context.Context, hash common.Hash) (*ethTypes.Header, error)
	FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]ethTypes.Log, error)
}

type resolvedGame struct {
	addr   common.Address
	l1Head common.Hash
}

// proofScanner finds the withdrawal proofs a game may have invalidated. A proof is only valid if it
// was submitted after the game was created, so scanning starts at the game's L1 head.
//
// Cursors are held in memory only. Losing them on restart just rescans from each game's L1 head.
type proofScanner struct {
	l1     L1Client
	portal Portal
	next   map[common.Address]uint64
}

func newProofScanner(l1 L1Client, portal Portal) *proofScanner {
	return &proofScanner{l1: l1, portal: portal, next: make(map[common.Address]uint64)}
}

func (s *proofScanner) Proofs(ctx context.Context, game resolvedGame, toBlock uint64) ([]contracts.WithdrawalProof, error) {
	from, ok := s.next[game.addr]
	if !ok {
		header, err := s.l1.HeaderByHash(ctx, game.l1Head)
		if err != nil {
			return nil, fmt.Errorf("failed to load l1 head %v: %w", game.l1Head, err)
		}
		from = header.Number.Uint64()
		s.next[game.addr] = from
	}
	if from > toBlock {
		// L1 reorged below the cursor, so rescan from the new head.
		from = toBlock
	}
	logs, err := s.l1.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{s.portal.Addr()},
		Topics:    [][]common.Hash{{s.portal.WithdrawalProvenExtension1Topic()}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch withdrawal proof logs from %v to %v: %w", from, toBlock, err)
	}
	proofs := make([]contracts.WithdrawalProof, 0, len(logs))
	seen := make(map[contracts.WithdrawalProof]bool, len(logs))
	for i := range logs {
		proof, err := s.portal.DecodeWithdrawalProvenExtension1(&logs[i])
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

// Advance records that every proof for game up to and including toBlock has been dealt with.
func (s *proofScanner) Advance(game common.Address, toBlock uint64) {
	s.next[game] = toBlock + 1
}

// Retain discards the cursors of games that are no longer being monitored.
func (s *proofScanner) Retain(games []common.Address) {
	retained := make(map[common.Address]uint64, len(games))
	for _, game := range games {
		if next, ok := s.next[game]; ok {
			retained[game] = next
		}
	}
	s.next = retained
}
