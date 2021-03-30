package rollup

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func isCtcTxEqual(a, b *types.Transaction) bool {
	if a.To() == nil && b.To() != nil {
		if !bytes.Equal(b.To().Bytes(), common.Address{}.Bytes()) {
			return false
		}
	}
	if a.To() != nil && b.To() == nil {
		if !bytes.Equal(a.To().Bytes(), common.Address{}.Bytes()) {
			return false
		}
		return false
	}
	if a.To() != nil && b.To() != nil {
		if !bytes.Equal(a.To().Bytes(), b.To().Bytes()) {
			return false
		}
	}
	if !bytes.Equal(a.Data(), b.Data()) {
		return false
	}
	if a.L1MessageSender() == nil && b.L1MessageSender() != nil {
		return false
	}
	if a.L1MessageSender() != nil && b.L1MessageSender() == nil {
		return false
	}
	if a.L1MessageSender() != nil && b.L1MessageSender() != nil {
		if !bytes.Equal(a.L1MessageSender().Bytes(), b.L1MessageSender().Bytes()) {
			return false
		}
	}
	if a.Gas() != b.Gas() {
		return false
	}
	return true
}
