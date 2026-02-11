package derive

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Network Upgrade Transactions (NUTs) are read from a JSON file and
// converted into deposit transactions.

// NUTMetadata contains version information for the NUT bundle format.
type NUTMetadata struct {
	Version string `json:"version"`
}

// NUTTransaction defines a single deposit transaction within a NUT bundle.
type NUTTransaction struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Data     hexutil.Bytes   `json:"data"`
	GasLimit uint64          `json:"gasLimit"`
	Value    *big.Int        `json:"value,omitempty"`
}

// NUTBundle is the top-level structure of a NUT file.
type NUTBundle struct {
	Metadata     NUTMetadata      `json:"metadata"`
	Transactions []NUTTransaction `json:"transactions"`
}

// ParseNUTBundle parses a NUT bundle from JSON bytes.
func ParseNUTBundle(data []byte) (*NUTBundle, error) {
	var bundle NUTBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse NUT bundle: %w", err)
	}
	return &bundle, nil
}

// ToDepositTransactions converts the bundle's transactions into serialized deposit transactions.
func (b *NUTBundle) ToDepositTransactions() ([]hexutil.Bytes, error) {
	txs := make([]hexutil.Bytes, 0, len(b.Transactions))
	for i, nutTx := range b.Transactions {
		value := nutTx.Value
		if value == nil {
			value = big.NewInt(0)
		}

		depTx := &types.DepositTx{
			// TODO: source hash derivation
			From:                nutTx.From,
			To:                  nutTx.To,
			Mint:                big.NewInt(0),
			Value:               value,
			Gas:                 nutTx.GasLimit,
			IsSystemTransaction: false,
			Data:                nutTx.Data,
		}

		encoded, err := types.NewTx(depTx).MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("tx %d: failed to marshal deposit tx: %w", i, err)
		}
		txs = append(txs, encoded)
	}
	return txs, nil
}
