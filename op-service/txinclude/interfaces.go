package txinclude

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

type Budget interface {
	Credit(eth.ETH)
	Debit(eth.ETH) error
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
