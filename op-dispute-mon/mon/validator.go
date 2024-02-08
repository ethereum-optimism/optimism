package mon

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type OutputRollupClient interface {
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type outputValidator struct {
	client OutputRollupClient
}

func newOutputValidator(client OutputRollupClient) *outputValidator {
	return &outputValidator{
		client: client,
	}
}

// CheckRootAgreement validates the specified root claim against the output at the given block number.
func (o *outputValidator) CheckRootAgreement(ctx context.Context, blockNum uint64, rootClaim common.Hash) (bool, common.Hash, error) {
	output, err := o.client.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return false, common.Hash{}, fmt.Errorf("failed to get output at block: %w", err)
	}
	expected := common.Hash(output.OutputRoot)
	return rootClaim == expected, expected, nil
}
