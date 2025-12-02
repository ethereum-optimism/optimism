package txinclude

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type Includer interface {
	Include(ctx context.Context, tx types.TxData) (*IncludedTx, error)
}

type IncludedTx struct {
	Transaction *types.Transaction
	Receipt     *types.Receipt
}

// EL represents an EVM execution layer.
// It is responsible for handling transport errors, such as HTTP 429s.
type EL interface {
	Sender
	ReceiptGetter
}

type ReceiptGetter interface {
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
}

type Sender interface {
	SendTransaction(context.Context, *types.Transaction) error
}

// Budget tracks costs throughout a tranaction's lifecycle.
type Budget interface {
	// BeforeResubmit is called before the transaction is resubmitted. It allows the cost to be
	// re-estimated in case of any changes (e.g., nonce increments affecting DA cost).
	BeforeResubmit(oldBudgetedCost eth.ETH, tx *types.Transaction) (newBudgetedCost eth.ETH, err error)
	// AfterCancel is called after the transaction is canceled, providing a chance to refund the
	// cost.
	AfterCancel(budgetedCost eth.ETH, tx *types.Transaction)
	// AfterIncluded is called after the transaction is included, giving an opportunity to refund
	// the difference between the budgeted cost and the actual cost.
	AfterIncluded(budgetedCost eth.ETH, tx *IncludedTx)
}

type UnlimitedBudget struct{}

var _ Budget = UnlimitedBudget{}

func (UnlimitedBudget) AfterCancel(_ eth.ETH, _ *types.Transaction) {}
func (UnlimitedBudget) AfterIncluded(_ eth.ETH, _ *IncludedTx)      {}
func (n UnlimitedBudget) BeforeResubmit(oldCost eth.ETH, _ *types.Transaction) (eth.ETH, error) {
	return oldCost, nil
}

type BasicBudget interface {
	Credit(eth.ETH) eth.ETH
	Debit(eth.ETH) (eth.ETH, error)
}

type ResubmitterObserver interface {
	SubmissionError(error)
}

type NoOpResubmitterObserver struct{}

var _ ResubmitterObserver = NoOpResubmitterObserver{}

func (NoOpResubmitterObserver) SubmissionError(error) {}

type Signer interface {
	Sign(context.Context, *types.Transaction) (*types.Transaction, error)
}

type PkSigner struct {
	pk      *ecdsa.PrivateKey
	chainID *big.Int
}

var _ Signer = (*PkSigner)(nil)

func (s *PkSigner) Sign(_ context.Context, tx *types.Transaction) (*types.Transaction, error) {
	return types.SignTx(tx, types.LatestSignerForChainID(s.chainID), s.pk)
}

func NewPkSigner(pk *ecdsa.PrivateKey, chainID *big.Int) *PkSigner {
	return &PkSigner{
		pk:      pk,
		chainID: chainID,
	}
}

type OPCostOracle interface {
	// OPCost returns the total OP-specific costs for tx, such as the L1 cost and operator cost.
	OPCost(*types.Transaction) *big.Int
}

type RPCClient interface {
	BatchCallContext(context.Context, []rpc.BatchElem) error
}
