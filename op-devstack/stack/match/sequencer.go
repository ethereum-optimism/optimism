package match

import (
	"context"

	"github.com/HashKeyChain/verse/op-devstack/stack"
	"github.com/HashKeyChain/verse/op-service/retry"
)

func WithSequencerActive(ctx context.Context) stack.Matcher[stack.L2CLNodeID, stack.L2CLNode] {
	return MatchElemFn[stack.L2CLNodeID, stack.L2CLNode](func(elem stack.L2CLNode) bool {
		sequencing, err := retry.Do(ctx, 10, retry.Exponential(), func() (bool, error) {
			return elem.RollupAPI().SequencerActive(ctx)
		})
		if err != nil {
			// Not available so can't be used by the test
			return false
		}
		return sequencing
	})
}
