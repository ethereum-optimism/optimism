package reviewperiod

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// LogFilterer reads logs from L1. ethclient.Client implements it.
type LogFilterer interface {
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
}

// logRangeSize keeps eth_getLogs requests within the block range limits of common RPC providers.
const logRangeSize = 10_000

// UpgradedLogs returns the Upgraded events emitted by the chain's proxies in blocks [from, to].
func UpgradedLogs(ctx context.Context, l1 LogFilterer, d ChainDelays, from, to uint64) ([]types.Log, error) {
	var addrs []common.Address
	for addr := range d.Proxies() {
		addrs = append(addrs, addr)
	}
	var out []types.Log
	for start := from; start <= to; start += logRangeSize {
		end := min(start+logRangeSize-1, to)
		logs, err := l1.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(start),
			ToBlock:   new(big.Int).SetUint64(end),
			Addresses: addrs,
			Topics:    [][]common.Hash{{UpgradedTopic}},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Upgraded logs for blocks %d-%d: %w", start, end, err)
		}
		out = append(out, logs...)
	}
	return out, nil
}

// CountUpgrades counts the Upgraded events emitted by the chain's proxies in logs, per contract name.
func CountUpgrades(d ChainDelays, logs []types.Log) map[string]int {
	proxies := d.Proxies()
	counts := make(map[string]int)
	for _, l := range logs {
		if name, ok := proxies[l.Address]; ok && len(l.Topics) > 0 && l.Topics[0] == UpgradedTopic {
			counts[name]++
		}
	}
	return counts
}

// DelayDiff is one delay's value before and after an upgrade.
type DelayDiff struct {
	Name          string
	Before, After uint64
}

// Changed reports whether the delay changed.
func (d DelayDiff) Changed() bool {
	return d.Before != d.After
}

// CompareDelays returns each of the three delays' values before and after an upgrade.
func CompareDelays(pre, post ChainDelays) []DelayDiff {
	return []DelayDiff{
		{"proofMaturityDelaySeconds", pre.ProofMaturityDelaySeconds, post.ProofMaturityDelaySeconds},
		{"disputeGameFinalityDelaySeconds", pre.DisputeGameFinalityDelaySeconds, post.DisputeGameFinalityDelaySeconds},
		{"withdrawalDelaySeconds", pre.WithdrawalDelaySeconds, post.WithdrawalDelaySeconds},
	}
}
