package sources

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ChainID retrieves the ChainID of an eth RPC directly.
// This helps identify the chain configuration to use for full typed bindings.
func ChainID(ctx context.Context, cl client.RPC) (eth.ChainID, error) {
	var id hexutil.Big
	err := cl.CallContext(ctx, &id, "eth_chainId")
	if err != nil {
		return eth.ChainID{}, err
	}
	return eth.ChainIDFromBig((*big.Int)(&id)), nil
}
