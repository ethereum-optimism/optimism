package txinclude

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Includer interface {
	Include(ctx context.Context, tx types.TxData) (*IncludedTx, error)
}

type IncludedTx struct {
	Transaction *types.Transaction
	Receipt     *optypes.Receipt
}

// EL represents an EVM execution layer.
// It is responsible for handling transport errors, such as HTTP 429s.
type EL interface {
	Sender
	ReceiptGetter
}

// NewReliableEL turns an implementation of the EL interface into one that will retry on
// intermittent failures.
func NewReliableEL(el EL, blockTime time.Duration) EL {
	return struct {
		*Monitor
		*Resubmitter
	}{
		NewMonitor(el, blockTime),
		NewResubmitter(el, blockTime),
	}
}

type ReceiptGetter interface {
	TransactionReceipt(context.Context, common.Hash) (*optypes.Receipt, error)
}

type Sender interface {
	SendTransaction(context.Context, *types.Transaction) error
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
