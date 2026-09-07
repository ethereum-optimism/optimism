package codec

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-private-interop/writes"
	"github.com/ethereum/go-ethereum/common"
)

var ErrWriteHistoryGap = errors.New("incomplete private write history")
var ErrStateChanged = errors.New("private state dependency changed")

// CheckUnchanged checks complete checkpoint-to-checkpoint write coverage. Callers
// must supply claims from the accepted canonical projection, with their configured
// chain/proof authentication already checked. Structural decoding is not authentication.
// Every storage dependency must include the account's existence and storage-reset tags.
// Dependencies are stable private tags; each claim uses its own public range tags.
// Forward gaps allowed by ClaimRegistry are NOT evidence that state stayed unchanged.
func CheckUnchanged(from, to uint64, dependencies []common.Hash, claims []*RangeClaim) error {
	if to < from {
		return ErrWriteHistoryGap
	}
	cursor := from
	for _, claim := range claims {
		if claim == nil || cursor == to || claim.FirstBlock != cursor+1 || claim.LastBlock > to {
			return ErrWriteHistoryGap
		}
		if err := claim.CheckStructure(); err != nil {
			return err
		}
		wanted := make(map[common.Hash]struct{}, len(dependencies))
		for _, tag := range dependencies {
			wanted[writes.RangeTag(claim.FirstBlock, claim.LastBlock, tag)] = struct{}{}
		}
		for _, r := range claim.Writes {
			if _, ok := wanted[r.Tag]; ok {
				return fmt.Errorf("%w: %s at block %d", ErrStateChanged, r.Tag, r.BlockNumber)
			}
		}
		cursor = claim.LastBlock
	}
	if cursor != to {
		return ErrWriteHistoryGap
	}
	return nil
}
