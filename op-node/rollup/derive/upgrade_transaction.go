package derive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/nuts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Network Upgrade Transactions (NUTs) are read from a JSON file and
// converted into deposit transactions.

// nutBundleVersion is the only bundle schema version this reader accepts.
const nutBundleVersion = "1.0.0"

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

// capitalizeForkName returns the fork name with its first character upper-cased.
// Mirrors rust/kona/crates/protocol/hardforks/build_helpers.rs::capitalize so the
// qualified intent strings (and therefore source hashes) agree across implementations.
func capitalizeForkName(f forks.Name) string {
	s := string(f)
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// readNUTBundle reads and parses a NUT bundle from an io.Reader. The fork name
// is used to namespace each transaction's intent when deriving source hashes.
func readNUTBundle(fork forks.Name, r io.Reader) (*nutBundle, error) {
	var bundle nutBundle
	if err := json.NewDecoder(r).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to parse NUT bundle: %w", err)
	}
	if bundle.Metadata.Version != nutBundleVersion {
		return nil, fmt.Errorf("unsupported NUT bundle version: got %q, want %q", bundle.Metadata.Version, nutBundleVersion)
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

		// The fork name is capitalized to match kona's NUT bundle codegen
		// (rust/kona/crates/protocol/hardforks/build_helpers.rs::capitalize),
		// so both implementations derive the same UpgradeDepositSource hashes.
		qualifiedIntent := fmt.Sprintf("%s %d: %s", capitalizeForkName(b.ForkName), i, nutTx.Intent)
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
	// bundleLabel is the concept-level identifier used to qualify intent
	// strings (and therefore source hashes). It is decoupled from the fork
	// name so a hard-fork rename (e.g., Interop → Lagoon) does not break
	// source-hash determinism with kona's bundle codegen, which embeds
	// the bundle's concept-level name ("interop" → "Interop").
	var bundleLabel forks.Name
	switch fork {
	case forks.Karst:
		bundleJSON = nuts.KarstNUTBundleJSON
		bundleLabel = forks.Karst
	case forks.Lagoon:
		bundleJSON = nuts.InteropNUTBundleJSON
		bundleLabel = "interop"
	default:
		return nil, 0, fmt.Errorf("no NUT bundle for fork %s", fork)
	}

	bundle, err := readNUTBundle(bundleLabel, bytes.NewReader(bundleJSON))
	if err != nil {
		return nil, 0, fmt.Errorf("reading %s NUT bundle: %w", fork, err)
	}

	txs, err := bundle.toDepositTransactions()
	if err != nil {
		return nil, 0, fmt.Errorf("converting %s NUT bundle to deposit txs: %w", fork, err)
	}

	return txs, bundle.totalGas(), nil
}
