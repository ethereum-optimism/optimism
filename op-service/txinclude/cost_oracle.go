package txinclude

import (
	"context"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	opfees "github.com/ethereum-optimism/optimism/op-core/fees"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
)

// CostOracle implements OPCostOracle, estimating the OP-specific costs of a transaction —
// the L1 data-availability fee and the operator fee — from parameters it fetches over RPC
// each block. It uses the Fjord L1 cost function and the Jovian operator-fee formula, the
// latter being exact on Jovian chains and a safe over-estimate on pre-Jovian ones (see
// opfees.OperatorCostJovian).
type CostOracle struct {
	client     RPCClient
	blockTime  time.Duration
	costParams atomic.Pointer[costParams]
}

type costParams struct {
	L1BaseFee           *big.Int
	L1BaseFeeScalar     *big.Int
	L1BlobBaseFee       *big.Int
	L1BlobBaseFeeScalar *big.Int
	OperatorFeeScalar   *big.Int
	OperatorFeeConstant *big.Int
}

var _ OPCostOracle = (*CostOracle)(nil)

func NewCostOracle(client RPCClient, blockTime time.Duration) *CostOracle {
	return &CostOracle{
		client:    client,
		blockTime: blockTime,
	}
}

func (i *CostOracle) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(i.blockTime):
			_ = i.SetParams(ctx) // Ignore error.
		}
	}
}

func (i *CostOracle) SetParams(ctx context.Context) error {
	batch := []rpc.BatchElem{
		newCall("basefee()"),
		newCall("baseFeeScalar()"),
		newCall("blobBaseFee()"),
		newCall("blobBaseFeeScalar()"),
		newCall("operatorFeeScalar()"),
		newCall("operatorFeeConstant()"),
	}
	if err := i.client.BatchCallContext(ctx, batch); err != nil {
		return fmt.Errorf("batch call: %w", err)
	}
	for _, elem := range batch {
		if elem.Error != nil {
			return fmt.Errorf("batch element error: %w", elem.Error)
		}
	}
	i.costParams.Store(&costParams{
		L1BaseFee:           new(big.Int).SetBytes(*batch[0].Result.(*hexutil.Bytes)),
		L1BaseFeeScalar:     new(big.Int).SetBytes(*batch[1].Result.(*hexutil.Bytes)),
		L1BlobBaseFee:       new(big.Int).SetBytes(*batch[2].Result.(*hexutil.Bytes)),
		L1BlobBaseFeeScalar: new(big.Int).SetBytes(*batch[3].Result.(*hexutil.Bytes)),
		OperatorFeeScalar:   new(big.Int).SetBytes(*batch[4].Result.(*hexutil.Bytes)),
		OperatorFeeConstant: new(big.Int).SetBytes(*batch[5].Result.(*hexutil.Bytes)),
	})
	return nil
}

func (i *CostOracle) OPCost(tx *types.Transaction) *big.Int {
	params := i.costParams.Load()

	l1CostFunc := types.NewL1CostFuncFjord(params.L1BaseFee, params.L1BlobBaseFee, params.L1BaseFeeScalar, params.L1BlobBaseFeeScalar)
	l1Cost, _ := l1CostFunc(tx.RollupCostData())

	operatorCost := opfees.OperatorCostJovian(tx.Gas(), bigs.Uint64Strict(params.OperatorFeeScalar), bigs.Uint64Strict(params.OperatorFeeConstant))

	return l1Cost.Add(l1Cost, operatorCost)
}

func newCall(method string) rpc.BatchElem {
	return rpc.BatchElem{
		Method: "eth_call",
		Args: []any{
			&signer.TransactionArgs{
				To:   ptr(common.HexToAddress(predeploys.L1Block)),
				Data: ptr(hexutil.Bytes(w3.MustNewFunc(method, "").Selector[:])),
			},
			eth.Unsafe,
			nil, // State overrides (optional).
		},
		Result: ptr(make(hexutil.Bytes, 0)),
	}
}

func ptr[T any](x T) *T {
	return &x
}
