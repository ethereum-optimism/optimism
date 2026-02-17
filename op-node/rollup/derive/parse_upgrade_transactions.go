package derive

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/forks"
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

// NetworkUpgradeTransaction defines a single deposit transaction within a NUT bundle.
type NetworkUpgradeTransaction struct {
	Intent   string          `json:"intent"`
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Data     hexutil.Bytes   `json:"data"`
	GasLimit uint64          `json:"gasLimit"`
}

// NUTBundle is the top-level structure of a NUT file.
type NUTBundle struct {
	ForkName     forks.Name                  `json:"-"`
	Metadata     NUTMetadata                 `json:"metadata"`
	Transactions []NetworkUpgradeTransaction `json:"transactions"`
}

// ReadNUTBundle reads and parses a NUT bundle from an io.Reader. The fork name
// is used to namespace each transaction's intent when deriving source hashes.
func ReadNUTBundle(fork forks.Name, r io.Reader) (*NUTBundle, error) {
	var bundle NUTBundle
	if err := json.NewDecoder(r).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to parse NUT bundle: %w", err)
	}
	bundle.ForkName = fork
	return &bundle, nil
}

// ToDepositTransactions converts the bundle's transactions into serialized deposit transactions.
func (b *NUTBundle) ToDepositTransactions() ([]hexutil.Bytes, error) {
	txs := make([]hexutil.Bytes, 0, len(b.Transactions))
	for i, nutTx := range b.Transactions {
		if nutTx.Intent == "" {
			return nil, fmt.Errorf("tx %d: missing intent", i)
		}

		qualifiedIntent := fmt.Sprintf("%s %d: %s", b.ForkName, i, nutTx.Intent)
		source := UpgradeDepositSource{Intent: qualifiedIntent}
		depTx := &types.DepositTx{
			SourceHash:          source.SourceHash(),
			From:                nutTx.From,
			To:                  nutTx.To,
			Mint:                big.NewInt(0),
			Value:               big.NewInt(0),
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
