package dsl

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// DefaultProofsCatchUpTimeout bounds how long a proofs-history store may lag behind the chain
// before a proof verification fails.
const DefaultProofsCatchUpTimeout = 30 * time.Second

// ProofsSyncStatus is the window of blocks an op-reth proofs-history store serves, as reported by
// debug_proofsSyncStatus. Both bounds are nil until the store has indexed its first block.
type ProofsSyncStatus struct {
	Earliest *uint64 `json:"earliest"`
	Latest   *uint64 `json:"latest"`
}

func (s ProofsSyncStatus) String() string {
	if s.Earliest == nil || s.Latest == nil {
		return "uninitialized"
	}
	return fmt.Sprintf("[%d, %d]", *s.Earliest, *s.Latest)
}

func (s ProofsSyncStatus) serves(block uint64) bool {
	return s.Earliest != nil && s.Latest != nil && *s.Earliest <= block && block <= *s.Latest
}

// ProofAccount names an account to prove, with the storage slots whose proofs are requested.
type ProofAccount struct {
	Address     common.Address
	StorageKeys []common.Hash
}

// HistoricalProofsOpts configures VerifyHistoricalProofsMatch.
type HistoricalProofsOpts struct {
	// Accounts are proven at every sampled block. L2ToL1MessagePasser is always proven and need
	// not be listed.
	Accounts []ProofAccount
	// CatchUpTimeout bounds how long the proofs store and the reference node may lag behind a
	// sampled block before the sweep fails.
	CatchUpTimeout time.Duration
	// FirstBlock and LastBlock bound the sweep. Zero selects the matching edge of the proofs
	// window reported by debug_proofsSyncStatus when the sweep starts.
	FirstBlock uint64
	LastBlock  uint64
	// MaxSampledBlocks bounds how many blocks of the range are compared; zero compares every
	// block. The first and last block of the range are always compared.
	MaxSampledBlocks int
}

// ProofsSyncStatus reports the window of blocks this node's proofs-history store serves.
func (el *L2ELNode) ProofsSyncStatus() (ProofsSyncStatus, error) {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	var status ProofsSyncStatus
	err := el.inner.L2EthClient().RPC().CallContext(ctx, &status, "debug_proofsSyncStatus")
	return status, err
}

// WaitForProofsStoreBlock waits until this node's proofs-history store serves target, so the RPCs
// answered from the store (eth_getProof, debug_executePayload, debug_executionWitness) can be
// called for it. The store indexes blocks asynchronously, so the chain head leads the store.
func (el *L2ELNode) WaitForProofsStoreBlock(target uint64, timeout time.Duration) ProofsSyncStatus {
	ctx, cancel := context.WithTimeout(el.ctx, timeout)
	defer cancel()
	logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "target", target, "timeout", timeout)
	logger.Info("Waiting for proofs store to index block")

	var lastStatus ProofsSyncStatus
	var lastErr error
	err := wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
		status, err := el.ProofsSyncStatus()
		if err != nil {
			lastErr = err
			logger.Warn("debug_proofsSyncStatus failed; will retry", "err", err)
			return false, nil
		}
		lastStatus, lastErr = status, nil
		if status.Latest != nil && *status.Latest >= target {
			logger.Info("Proofs store indexed block", "window", status)
			return true, nil
		}
		logger.Info("Proofs store behind target", "window", status)
		return false, nil
	})
	el.require.NoErrorf(err, "proofs store on %s did not index block %d within %s; last window %s; last error: %v",
		el.inner.Name(), target, timeout, lastStatus, lastErr)
	return lastStatus
}

// VerifyHistoricalProofsMatch sweeps the blocks this node's proofs-history store serves and, at
// every sampled block, requires eth_getProof for each configured account to verify against the
// block's state root and to equal the proof served by reference. The L2ToL1MessagePasser proof is
// always included, and its storage root must equal the header's withdrawalsRoot where the header
// carries one.
//
// reference must answer eth_getProof from its own historical state rather than from a
// proofs-history store, since two stores can agree on a wrong proof.
func (el *L2ELNode) VerifyHistoricalProofsMatch(reference *L2ELNode, opts ...func(*HistoricalProofsOpts)) {
	cfg := HistoricalProofsOpts{CatchUpTimeout: DefaultProofsCatchUpTimeout}
	for _, opt := range opts {
		opt(&cfg)
	}
	logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "reference", reference.inner.Name())

	el.requireIndependentProofReference(reference)

	window := el.WaitForProofsStoreBlock(cfg.LastBlock, cfg.CatchUpTimeout)
	first, last := cfg.FirstBlock, cfg.LastBlock
	if first == 0 {
		first = *window.Earliest
	}
	if last == 0 {
		last = *window.Latest
	}
	el.require.LessOrEqualf(first, last, "historical proofs sweep range [%d, %d] is empty; proofs window %s", first, last, window)
	blocks := sampleBlockNumbers(first, last, cfg.MaxSampledBlocks)
	accounts := append([]ProofAccount{{Address: predeploys.L2ToL1MessagePasserAddr}}, cfg.Accounts...)

	logger.Info("Sweeping historical proofs", "window", window, "first", first, "last", last,
		"blocks", len(blocks), "accounts", len(accounts))
	for _, block := range blocks {
		el.verifyProofsMatchAt(reference, block, accounts, cfg.CatchUpTimeout)
	}
	logger.Info("Historical proofs match the reference", "first", first, "last", last,
		"blocks", len(blocks), "accounts", len(accounts))
}

// VerifyProofsMatchAt is VerifyHistoricalProofsMatch for one block and an explicit account list,
// waiting up to DefaultProofsCatchUpTimeout for the proofs store to serve the block.
func (el *L2ELNode) VerifyProofsMatchAt(reference *L2ELNode, block uint64, accounts ...ProofAccount) {
	el.WaitForProofsStoreBlock(block, DefaultProofsCatchUpTimeout)
	el.verifyProofsMatchAt(reference, block, accounts, DefaultProofsCatchUpTimeout)
}

func (el *L2ELNode) verifyProofsMatchAt(reference *L2ELNode, block uint64, accounts []ProofAccount, catchUpTimeout time.Duration) {
	window, err := el.ProofsSyncStatus()
	el.require.NoErrorf(err, "debug_proofsSyncStatus failed on %s", el.inner.Name())
	el.require.Truef(window.serves(block), "proofs store on %s does not serve block %d: window %s", el.inner.Name(), block, window)

	header := el.awaitHeaderByNumber(block, catchUpTimeout)
	referenceHeader := reference.awaitHeaderByNumber(block, catchUpTimeout)
	el.require.Equalf(referenceHeader.Hash(), header.Hash(), "block %d: %s and reference %s disagree on the block hash",
		block, el.inner.Name(), reference.inner.Name())

	for _, account := range accounts {
		el.verifyAccountProofMatchesAt(reference, header, account, catchUpTimeout)
	}
	el.log.Info("Historical proofs verified", "name", el.inner.Name(), "block", block, "accounts", len(accounts))
}

func (el *L2ELNode) verifyAccountProofMatchesAt(reference *L2ELNode, header eth.BlockInfo, account ProofAccount, catchUpTimeout time.Duration) {
	block := header.NumberU64()
	proof := el.awaitProof(header, account, catchUpTimeout)
	referenceProof := reference.awaitProof(header, account, catchUpTimeout)
	NormalizeProofResponse(proof)
	NormalizeProofResponse(referenceProof)

	el.require.NoErrorf(VerifyAccountProof(proof, header.Root()),
		"block %d account %s: eth_getProof from %s does not verify against state root %s",
		block, account.Address, el.inner.Name(), header.Root())
	el.require.NoErrorf(VerifyAccountProof(referenceProof, header.Root()),
		"block %d account %s: eth_getProof from reference %s does not verify against state root %s",
		block, account.Address, reference.inner.Name(), header.Root())
	el.require.Equalf(referenceProof, proof, "block %d account %s: eth_getProof from %s differs from reference %s",
		block, account.Address, el.inner.Name(), reference.inner.Name())

	if account.Address == predeploys.L2ToL1MessagePasserAddr && header.WithdrawalsRoot() != nil {
		el.require.Equalf(*header.WithdrawalsRoot(), proof.StorageHash,
			"block %d: L2ToL1MessagePasser storage root from %s does not equal the header's withdrawalsRoot", block, el.inner.Name())
	}
	el.log.Debug("Historical proof matches the reference", "name", el.inner.Name(), "block", block,
		"account", account.Address, "storage_keys", len(account.StorageKeys))
}

// requireIndependentProofReference fails when reference serves eth_getProof from a proofs-history
// store, which every node that exposes a proofs window does.
func (el *L2ELNode) requireIndependentProofReference(reference *L2ELNode) {
	status, err := reference.ProofsSyncStatus()
	if err != nil {
		el.log.Debug("Reference node has no proofs-history store", "reference", reference.inner.Name(), "err", err)
		return
	}
	el.require.Nilf(status.Latest, "reference %s runs proofs history (window %s); the sweep needs a reference that serves eth_getProof from its own historical state",
		reference.inner.Name(), status)
}

func (el *L2ELNode) awaitHeaderByNumber(number uint64, timeout time.Duration) eth.BlockInfo {
	ctx, cancel := context.WithTimeout(el.ctx, timeout)
	defer cancel()
	var header eth.BlockInfo
	var lastErr error
	err := wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
		info, err := el.inner.L2EthClient().InfoByNumber(ctx, number)
		if err != nil {
			lastErr = err
			el.log.Debug("Header lookup failed; will retry", "name", el.inner.Name(), "block", number, "err", err)
			return false, nil
		}
		header = info
		return true, nil
	})
	el.require.NoErrorf(err, "%s did not serve the header of block %d within %s; last error: %v", el.inner.Name(), number, timeout, lastErr)
	return header
}

// awaitProof fetches eth_getProof by block hash, so a block that leaves the canonical chain while
// the sweep runs fails the lookup rather than returning the proof of its replacement.
func (el *L2ELNode) awaitProof(header eth.BlockInfo, account ProofAccount, timeout time.Duration) *eth.AccountResult {
	ctx, cancel := context.WithTimeout(el.ctx, timeout)
	defer cancel()
	// A nil key list encodes as JSON null, which op-reth rejects where op-geth accepts it.
	storageKeys := account.StorageKeys
	if storageKeys == nil {
		storageKeys = []common.Hash{}
	}
	var proof *eth.AccountResult
	var lastErr error
	err := wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
		result, err := el.inner.L2EthClient().GetProof(ctx, account.Address, storageKeys, header.Hash().String())
		if err != nil {
			lastErr = err
			el.log.Warn("eth_getProof failed; will retry", "name", el.inner.Name(), "block", header.NumberU64(),
				"account", account.Address, "err", err)
			return false, nil
		}
		proof = result
		return true, nil
	})
	el.require.NoErrorf(err, "%s did not serve eth_getProof for account %s at block %d within %s; last error: %v",
		el.inner.Name(), account.Address, header.NumberU64(), timeout, lastErr)
	return proof
}

// sampleBlockNumbers returns at most maxSamples evenly spaced block numbers in [first, last],
// always including both edges. A non-positive maxSamples returns every block.
func sampleBlockNumbers(first, last uint64, maxSamples int) []uint64 {
	count := last - first + 1
	if maxSamples <= 0 || uint64(maxSamples) >= count {
		blocks := make([]uint64, 0, count)
		for block := first; block <= last; block++ {
			blocks = append(blocks, block)
		}
		return blocks
	}
	if maxSamples < 2 {
		maxSamples = 2
	}
	blocks := make([]uint64, 0, maxSamples)
	for i := range maxSamples {
		blocks = append(blocks, first+(count-1)*uint64(i)/uint64(maxSamples-1))
	}
	return blocks
}

// NormalizeProofResponse rewrites an eth_getProof response into one canonical encoding so proofs
// from different clients compare equal: an empty storage proof, which reth encodes as ["0x80"],
// becomes empty; a zero code hash becomes keccak256("") and a zero storage hash becomes the empty
// trie root, which is how reth encodes both; storage keys are left-padded to 32 bytes; and an
// absent storage proof list becomes nil.
func NormalizeProofResponse(res *eth.AccountResult) {
	for i := range res.StorageProof {
		if len(res.StorageProof[i].Proof) == 1 && bytes.Equal(res.StorageProof[i].Proof[0], []byte{0x80}) {
			res.StorageProof[i].Proof = []hexutil.Bytes{}
		}
		res.StorageProof[i].Key = common.BytesToHash(res.StorageProof[i].Key).Bytes()
	}
	if len(res.StorageProof) == 0 {
		res.StorageProof = nil
	}
	if res.CodeHash == (common.Hash{}) {
		res.CodeHash = crypto.Keccak256Hash(nil)
	}
	if res.StorageHash == (common.Hash{}) {
		res.StorageHash = types.EmptyRootHash
	}
}

// VerifyAccountProof verifies an eth_getProof response against a state root. Unlike
// eth.AccountResult.Verify it accepts an empty storage trie, whose proofs carry no nodes, and a
// non-existence proof for an account the response reports as empty.
func VerifyAccountProof(res *eth.AccountResult, stateRoot common.Hash) error {
	if res.StorageHash != types.EmptyRootHash {
		for i, entry := range res.StorageProof {
			db := memorydb.New()
			for j, encodedNode := range entry.Proof {
				nodeKey := encodedNode
				if len(encodedNode) >= 32 { // small MPT nodes are not hashed
					nodeKey = crypto.Keccak256(encodedNode)
				}
				if err := db.Put(nodeKey, encodedNode); err != nil {
					return fmt.Errorf("failed to load storage proof node %d of storage value %d into mem db: %w", j, i, err)
				}
			}
			path := crypto.Keccak256(entry.Key)
			val, err := trie.VerifyProof(res.StorageHash, path, db)
			if err != nil {
				return fmt.Errorf("failed to verify storage value %d with key %s (path %x) in storage trie %s: %w", i, entry.Key.String(), path, res.StorageHash, err)
			}
			if val == nil && entry.Value.ToInt().Cmp(common.Big0) == 0 { // empty storage is zero by default
				continue
			}
			comparison, err := rlp.EncodeToBytes(entry.Value.ToInt().Bytes())
			if err != nil {
				return fmt.Errorf("failed to encode storage value %d with key %s (path %x) in storage trie %s: %w", i, entry.Key.String(), path, res.StorageHash, err)
			}
			if !bytes.Equal(val, comparison) {
				return fmt.Errorf("value %d in storage proof does not match proven value at key %s (path %x)", i, entry.Key.String(), path)
			}
		}
	}

	accountClaimed := []any{uint64(res.Nonce), res.Balance.ToInt().Bytes(), res.StorageHash, res.CodeHash}
	accountClaimedValue, err := rlp.EncodeToBytes(accountClaimed)
	if err != nil {
		return fmt.Errorf("failed to encode account from retrieved values: %w", err)
	}

	db := memorydb.New()
	for i, encodedNode := range res.AccountProof {
		nodeKey := encodedNode
		if len(encodedNode) >= 32 { // small MPT nodes are not hashed
			nodeKey = crypto.Keccak256(encodedNode)
		}
		if err := db.Put(nodeKey, encodedNode); err != nil {
			return fmt.Errorf("failed to load account proof node %d into mem db: %w", i, err)
		}
	}
	path := crypto.Keccak256(res.Address[:])
	accountProofValue, err := trie.VerifyProof(stateRoot, path, db)
	if err != nil {
		return fmt.Errorf("failed to verify account value with key %s (path %x) in account trie %s: %w", res.Address, path, stateRoot, err)
	}

	// A non-existence proof is only valid when the response reports an empty account.
	if len(accountProofValue) == 0 {
		isEmpty := res.Nonce == 0 &&
			res.Balance.ToInt().Sign() == 0 &&
			(res.StorageHash == types.EmptyRootHash || res.StorageHash == common.Hash{}) &&
			(res.CodeHash == crypto.Keccak256Hash(nil) || res.CodeHash == common.Hash{})
		if isEmpty {
			return nil
		}
	}

	if !bytes.Equal(accountClaimedValue, accountProofValue) {
		return fmt.Errorf("account proof does not match the claimed account values:\n"+
			"  claimed: %x\n"+
			"  proof:   %x", accountClaimedValue, accountProofValue)
	}
	return nil
}
