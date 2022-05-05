package sequencer

import (
	"bytes"
	"fmt"

	l2types "github.com/ethereum-optimism/optimism/l2geth/core/types"
)

type CachedTx struct {
	tx    *l2types.Transaction
	rawTx []byte
}

func NewCachedTx(tx *l2types.Transaction) *CachedTx {
	var txBuf bytes.Buffer
	if err := tx.EncodeRLP(&txBuf); err != nil {
		panic(fmt.Sprintf("Unable to encode tx: %v", err))
	}

	return &CachedTx{
		tx:    tx,
		rawTx: txBuf.Bytes(),
	}
}

func (t *CachedTx) Tx() *l2types.Transaction {
	return t.tx
}

func (t *CachedTx) Size() int {
	return len(t.rawTx)
}

func (t *CachedTx) RawTx() []byte {
	return t.rawTx
}
