package match

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

func WithSequencerActive(ctx context.Context) stack.Matcher[stack.L2CLNodeID, stack.L2CLNode] {
	return MatchElemFn[stack.L2CLNodeID, stack.L2CLNode](func(elem stack.L2CLNode) bool {
		sequencing, err := retry.Do(ctx, 10, retry.Exponential(), func() (bool, error) {
			res, err := elem.RollupAPI().SequencerActive(ctx)
			fmt.Printf("1237774 %v %v %v\n", res, err, elem.ID())
			if err != nil && strings.Contains(err.Error(), "Method not found") {
				fmt.Println("1237774 swirl")
				return false, nil
			}
			return res, err
		})
		if err != nil {
			// Not available so can't be used by the test
			return false
		}
		return sequencing
	})
}
