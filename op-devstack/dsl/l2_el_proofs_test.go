package dsl

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/stretchr/testify/require"
)

// provenAccount is one account of a synthetic state trie. Zero-valued slots are left out of the
// storage trie, as a client would.
type provenAccount struct {
	address common.Address
	balance *big.Int
	nonce   uint64
	slots   map[common.Hash]*big.Int
}

var (
	alice = provenAccount{
		address: common.HexToAddress("0xa11ce"),
		balance: big.NewInt(1_000),
		nonce:   3,
		slots: map[common.Hash]*big.Int{
			common.HexToHash("0x0"): big.NewInt(42),
			common.HexToHash("0x1"): big.NewInt(0),
		},
	}
	bob = provenAccount{
		address: common.HexToAddress("0xb0b"),
		balance: big.NewInt(7),
		nonce:   1,
	}
	carol = provenAccount{
		address: common.HexToAddress("0xca201"),
		balance: big.NewInt(99),
		nonce:   0,
		slots:   map[common.Hash]*big.Int{common.HexToHash("0x5"): big.NewInt(5)},
	}
	nobody = provenAccount{address: common.HexToAddress("0x1234"), balance: big.NewInt(0)}
)

func newTestTrie() *trie.Trie {
	return trie.NewEmpty(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil))
}

func proveKey(t *testing.T, tr *trie.Trie, key []byte) []hexutil.Bytes {
	var nodes trienode.ProofList
	require.NoError(t, tr.Prove(crypto.Keccak256(key), &nodes))
	proof := make([]hexutil.Bytes, len(nodes))
	for i, node := range nodes {
		proof[i] = hexutil.Bytes(node)
	}
	return proof
}

func storageTrieOf(t *testing.T, account provenAccount) *trie.Trie {
	storageTrie := newTestTrie()
	for slot, value := range account.slots {
		if value.Sign() == 0 {
			continue
		}
		encoded, err := rlp.EncodeToBytes(value.Bytes())
		require.NoError(t, err)
		storageTrie.MustUpdate(crypto.Keccak256(slot.Bytes()), encoded)
	}
	return storageTrie
}

// stateTrieOf builds a state trie holding every account in accounts.
func stateTrieOf(t *testing.T, accounts ...provenAccount) *trie.Trie {
	stateTrie := newTestTrie()
	for _, account := range accounts {
		encoded, err := rlp.EncodeToBytes([]any{account.nonce, account.balance.Bytes(), storageTrieOf(t, account).Hash(), crypto.Keccak256Hash(nil)})
		require.NoError(t, err)
		stateTrie.MustUpdate(crypto.Keccak256(account.address.Bytes()), encoded)
	}
	return stateTrie
}

// proofOf returns what eth_getProof would answer for account against stateTrie, whether or not
// the trie holds the account.
func proofOf(t *testing.T, stateTrie *trie.Trie, account provenAccount) *eth.AccountResult {
	storageTrie := storageTrieOf(t, account)
	result := &eth.AccountResult{
		AccountProof: proveKey(t, stateTrie, account.address.Bytes()),
		Address:      account.address,
		Balance:      (*hexutil.Big)(account.balance),
		CodeHash:     crypto.Keccak256Hash(nil),
		Nonce:        hexutil.Uint64(account.nonce),
		StorageHash:  storageTrie.Hash(),
	}
	for slot, value := range account.slots {
		result.StorageProof = append(result.StorageProof, eth.StorageProofEntry{
			Key:   slot.Bytes(),
			Value: (hexutil.Big)(*value),
			Proof: proveKey(t, storageTrie, slot.Bytes()),
		})
	}
	return result
}

func TestVerifyAccountProof(t *testing.T) {
	stateTrie := stateTrieOf(t, alice, bob, carol)
	stateRoot := stateTrie.Hash()

	for _, tc := range []struct {
		name    string
		proof   func() *eth.AccountResult
		root    common.Hash
		wantErr string
	}{
		{name: "account with storage verifies", proof: func() *eth.AccountResult { return proofOf(t, stateTrie, alice) }, root: stateRoot},
		{name: "account without storage verifies", proof: func() *eth.AccountResult { return proofOf(t, stateTrie, bob) }, root: stateRoot},
		{name: "absent account reported empty verifies", proof: func() *eth.AccountResult { return proofOf(t, stateTrie, nobody) }, root: stateRoot},
		{
			name: "zero slot of an empty storage trie verifies",
			proof: func() *eth.AccountResult {
				proof := proofOf(t, stateTrie, bob)
				proof.StorageProof = []eth.StorageProofEntry{{Key: common.HexToHash("0x0").Bytes(), Proof: []hexutil.Bytes{}}}
				return proof
			},
			root: stateRoot,
		},
		{
			name: "tampered balance fails",
			proof: func() *eth.AccountResult {
				proof := proofOf(t, stateTrie, alice)
				proof.Balance = (*hexutil.Big)(big.NewInt(1_001))
				return proof
			},
			root:    stateRoot,
			wantErr: "account proof does not match the claimed account values",
		},
		{
			name: "tampered storage value fails",
			proof: func() *eth.AccountResult {
				proof := proofOf(t, stateTrie, carol)
				proof.StorageProof[0].Value = (hexutil.Big)(*big.NewInt(6))
				return proof
			},
			root:    stateRoot,
			wantErr: "does not match proven value",
		},
		{
			name: "nonzero slot of an empty storage trie fails",
			proof: func() *eth.AccountResult {
				proof := proofOf(t, stateTrie, bob)
				proof.StorageProof = []eth.StorageProofEntry{{Key: common.HexToHash("0x0").Bytes(), Value: (hexutil.Big)(*big.NewInt(42)), Proof: []hexutil.Bytes{}}}
				return proof
			},
			root:    stateRoot,
			wantErr: "is nonzero but the storage trie is empty",
		},
		{
			name: "absent account claiming a balance fails",
			proof: func() *eth.AccountResult {
				proof := proofOf(t, stateTrie, nobody)
				proof.Balance = (*hexutil.Big)(big.NewInt(1))
				return proof
			},
			root:    stateRoot,
			wantErr: "account proof does not match the claimed account values",
		},
		{
			name:    "wrong state root fails",
			proof:   func() *eth.AccountResult { return proofOf(t, stateTrie, alice) },
			root:    common.HexToHash("0xdead"),
			wantErr: "failed to verify account value",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifyAccountProof(tc.proof(), tc.root)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestNormalizeProofResponse(t *testing.T) {
	emptyCodeHash := crypto.Keccak256Hash(nil)
	someHash := common.HexToHash("0xabc")
	key := common.HexToHash("0x1").Bytes()

	for _, tc := range []struct {
		name string
		in   eth.AccountResult
		want eth.AccountResult
	}{
		{
			name: "reth empty storage proof becomes empty",
			in:   eth.AccountResult{CodeHash: someHash, StorageHash: someHash, StorageProof: []eth.StorageProofEntry{{Key: key, Proof: []hexutil.Bytes{{0x80}}}}},
			want: eth.AccountResult{CodeHash: someHash, StorageHash: someHash, StorageProof: []eth.StorageProofEntry{{Key: key, Proof: []hexutil.Bytes{}}}},
		},
		{
			name: "zero code hash becomes the empty code hash",
			in:   eth.AccountResult{StorageHash: someHash},
			want: eth.AccountResult{CodeHash: emptyCodeHash, StorageHash: someHash},
		},
		{
			name: "zero storage hash becomes the empty trie root",
			in:   eth.AccountResult{CodeHash: someHash},
			want: eth.AccountResult{CodeHash: someHash, StorageHash: types.EmptyRootHash},
		},
		{
			name: "empty storage proof list becomes nil",
			in:   eth.AccountResult{CodeHash: someHash, StorageHash: someHash, StorageProof: []eth.StorageProofEntry{}},
			want: eth.AccountResult{CodeHash: someHash, StorageHash: someHash},
		},
		{
			name: "non-empty values are kept",
			in:   eth.AccountResult{CodeHash: someHash, StorageHash: someHash, StorageProof: []eth.StorageProofEntry{{Key: key, Proof: []hexutil.Bytes{{0x80}, {0x81}}}}},
			want: eth.AccountResult{CodeHash: someHash, StorageHash: someHash, StorageProof: []eth.StorageProofEntry{{Key: key, Proof: []hexutil.Bytes{{0x80}, {0x81}}}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			NormalizeProofResponse(&got)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestProofsSyncStatusServes(t *testing.T) {
	window := func(earliest, latest uint64) ProofsSyncStatus {
		return ProofsSyncStatus{Earliest: &earliest, Latest: &latest}
	}
	for _, tc := range []struct {
		name   string
		status ProofsSyncStatus
		block  uint64
		want   bool
	}{
		{name: "uninitialized store serves nothing", status: ProofsSyncStatus{}, block: 0, want: false},
		{name: "earliest edge is served", status: window(5, 9), block: 5, want: true},
		{name: "latest edge is served", status: window(5, 9), block: 9, want: true},
		{name: "block below earliest is not served", status: window(5, 9), block: 4, want: false},
		{name: "block above latest is not served", status: window(5, 9), block: 10, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.status.serves(tc.block))
		})
	}
}

func TestSampleBlockNumbers(t *testing.T) {
	for _, tc := range []struct {
		name       string
		first      uint64
		last       uint64
		maxSamples int
		want       []uint64
	}{
		{name: "zero bound returns every block", first: 0, last: 4, maxSamples: 0, want: []uint64{0, 1, 2, 3, 4}},
		{name: "single block", first: 5, last: 5, maxSamples: 0, want: []uint64{5}},
		{name: "bound above count returns every block", first: 0, last: 4, maxSamples: 20, want: []uint64{0, 1, 2, 3, 4}},
		{name: "bound of two keeps the edges", first: 0, last: 9, maxSamples: 2, want: []uint64{0, 9}},
		{name: "bound of one still keeps both edges", first: 0, last: 9, maxSamples: 1, want: []uint64{0, 9}},
		{name: "samples are evenly spaced", first: 0, last: 100, maxSamples: 5, want: []uint64{0, 25, 50, 75, 100}},
		{name: "uneven spacing rounds down and keeps the last block", first: 10, last: 13, maxSamples: 3, want: []uint64{10, 11, 13}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, sampleBlockNumbers(tc.first, tc.last, tc.maxSamples))
		})
	}
}
