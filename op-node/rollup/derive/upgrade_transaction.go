package derive

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:embed karst_nut_bundle.json
var karstNUTBundleJSON []byte

// Network Upgrade Transactions (NUTs) are read from a JSON file and
// converted into deposit transactions.

// nutMetadata contains version information for the NUT bundle format.
type nutMetadata struct {
	Version string `json:"version"`
}

// networkUpgradeTransaction defines a single deposit transaction within a NUT bundle.
type networkUpgradeTransaction struct {
	Intent   string          `json:"intent"`
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Data     hexutil.Bytes   `json:"data"`
	GasLimit uint64          `json:"gasLimit"`
}

// nutBundle is the top-level structure of a NUT file.
type nutBundle struct {
	ForkName     forks.Name                  `json:"-"`
	Metadata     nutMetadata                 `json:"metadata"`
	Transactions []networkUpgradeTransaction `json:"transactions"`
}

// readNUTBundle reads and parses a NUT bundle from an io.Reader. The fork name
// is used to namespace each transaction's intent when deriving source hashes.
func readNUTBundle(fork forks.Name, r io.Reader) (*nutBundle, error) {
	var bundle nutBundle
	if err := json.NewDecoder(r).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to parse NUT bundle: %w", err)
	}
	bundle.ForkName = fork
	return &bundle, nil
}

// totalGas returns the sum of gas limits across all transactions in the bundle.
func (b *nutBundle) totalGas() uint64 {
	var total uint64
	for _, tx := range b.Transactions {
		total += tx.GasLimit
	}
	return total
}

// toDepositTransactions converts the bundle's transactions into serialized deposit transactions.
func (b *nutBundle) toDepositTransactions() ([]hexutil.Bytes, error) {
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

// UpgradeTransactions returns the deposit transactions and total gas required for a
// fork's NUT bundle. The fork name selects the embedded bundle JSON.
func UpgradeTransactions(fork forks.Name) ([]hexutil.Bytes, uint64, error) {
	var bundleJSON []byte
	switch fork {
	case forks.Karst:
		bundleJSON = karstNUTBundleJSON
	default:
		return nil, 0, fmt.Errorf("no NUT bundle for fork %s", fork)
	}

	bundle, err := readNUTBundle(fork, bytes.NewReader(bundleJSON))
	if err != nil {
		return nil, 0, fmt.Errorf("reading %s NUT bundle: %w", fork, err)
	}

	txs, err := bundle.toDepositTransactions()
	if err != nil {
		return nil, 0, fmt.Errorf("converting %s NUT bundle to deposit txs: %w", fork, err)
	}

	return txs, bundle.totalGas(), nil
}
