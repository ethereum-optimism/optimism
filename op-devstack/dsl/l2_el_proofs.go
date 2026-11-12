package dsl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
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

func (s ProofsSyncStatus) prunedPast(block uint64) bool {
	return s.Earliest != nil && block < *s.Earliest
}

// ProofAccount names an account to prove, with the storage slots whose proofs are requested.
type ProofAccount struct {
	Address     common.Address
	StorageKeys []common.Hash
}

// HistoricalProofsOpts configures VerifyHistoricalProofsMatch.
type HistoricalProofsOpts struct {
	// Accounts are proven at every sampled block. L2ToL1MessagePasser is proven whether or not it
	// is listed.
	Accounts []ProofAccount
	// CatchUpTimeout bounds how long the proofs store and the reference node may lag behind a
	// sampled block, and how long each read of the proofs window may keep failing, before the
	// sweep fails. Zero selects DefaultProofsCatchUpTimeout.
	CatchUpTimeout time.Duration
	// FirstBlock and LastBlock bound the sweep. Zero selects the matching edge of the proofs
	// window reported by debug_proofsSyncStatus when the sweep starts. A block the store prunes
	// while the sweep runs is skipped when FirstBlock is zero and fails the sweep otherwise. The
	// sweep fails if it skips every sampled block.
	FirstBlock uint64
	LastBlock  uint64
	// MaxSampledBlocks bounds how many blocks of the range are compared; zero compares every
	// block. The first and last block of the range are always compared, so at least two blocks
	// of a multi-block range are.
	MaxSampledBlocks int
}

// ProofsSyncStatus reports the window of blocks this node's proofs-history store serves. The call is
// bounded by ctx and by DefaultTimeout, whichever ends first.
func (el *L2ELNode) ProofsSyncStatus(ctx context.Context) (ProofsSyncStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	var status ProofsSyncStatus
	err := el.inner.L2EthClient().RPC().CallContext(ctx, &status, "debug_proofsSyncStatus")
	return status, err
}

// WaitForProofsStoreBlock waits until this node's proofs-history store serves target, so the RPCs
// answered from the store (eth_getProof, debug_executePayload, debug_executionWitness) can be
// called for it. The store indexes blocks asynchronously, so the chain head leads the store.
func (el *L2ELNode) WaitForProofsStoreBlock(target uint64, timeout time.Duration) ProofsSyncStatus {
	logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "target", target, "timeout", timeout)
	logger.Info("Waiting for proofs store to index block")
	return el.pollProofsSyncStatus(timeout, fmt.Sprintf("index block %d", target), func(status ProofsSyncStatus) bool {
		if status.Earliest != nil && status.Latest != nil && *status.Latest >= target {
			logger.Info("Proofs store indexed block", "window", status)
			return true
		}
		logger.Info("Proofs store behind target", "window", status)
		return false
	})
}

// pollProofsSyncStatus retries debug_proofsSyncStatus until done accepts the window it reports,
// failing once timeout ends with a message saying the store did not reach goal.
func (el *L2ELNode) pollProofsSyncStatus(timeout time.Duration, goal string, done func(ProofsSyncStatus) bool) ProofsSyncStatus {
	ctx, cancel := context.WithTimeout(el.ctx, timeout)
	defer cancel()
	var lastStatus ProofsSyncStatus
	var lastErr error
	err := wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
		status, err := el.ProofsSyncStatus(ctx)
		if err != nil {
			lastErr = err
			el.log.Warn("debug_proofsSyncStatus failed; will retry", "name", el.inner.Name(), "goal", goal,
				"timeout", timeout, "err", err)
			return false, nil
		}
		lastStatus, lastErr = status, nil
		return done(status), nil
	})
	el.require.NoErrorf(err, "proofs store on %s did not %s within %s; last window %s; last error: %v",
		el.inner.Name(), goal, timeout, lastStatus, lastErr)
	return lastStatus
}

// VerifyHistoricalProofsMatch sweeps the blocks this node's proofs-history store serves and, at
// every sampled block, requires eth_getProof for each configured account to verify against the
// block's state root and to equal the proof served by reference. The L2ToL1MessagePasser proof is
// always included, and from Isthmus on its storage root must equal the header's withdrawalsRoot.
//
// reference must answer eth_getProof from its own historical state rather than from a
// proofs-history store, since two stores can agree on a wrong proof.
func (el *L2ELNode) VerifyHistoricalProofsMatch(reference *L2ELNode, opts ...func(*HistoricalProofsOpts)) {
	var cfg HistoricalProofsOpts
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.CatchUpTimeout == 0 {
		cfg.CatchUpTimeout = DefaultProofsCatchUpTimeout
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
	accounts := withMessagePasser(cfg.Accounts)

	logger.Info("Sweeping historical proofs", "window", window, "first", first, "last", last,
		"blocks", len(blocks), "accounts", len(accounts))
	skipPrunedBlocks := cfg.FirstBlock == 0
	pruned := 0
	for _, block := range blocks {
		window := el.awaitProofsSyncStatus(cfg.CatchUpTimeout)
		if skipPrunedBlocks && window.prunedPast(block) {
			logger.Info("Block left the proofs window during the sweep; skipping", "block", block, "window", window)
			pruned++
			continue
		}
		if el.verifyProofsMatchAt(reference, block, window, accounts, cfg.CatchUpTimeout, skipPrunedBlocks) {
			pruned++
		}
	}
	verified := len(blocks) - pruned
	el.require.Positivef(verified, "historical proofs sweep on %s verified no block: all %d sampled blocks of [%d, %d] left the proofs window during the sweep",
		el.inner.Name(), len(blocks), first, last)
	logger.Info("Historical proofs match the reference", "first", first, "last", last,
		"blocks", verified, "pruned_during_sweep", pruned, "accounts", len(accounts))
}

// withMessagePasser returns accounts with L2ToL1MessagePasser prepended unless it is already listed.
func withMessagePasser(accounts []ProofAccount) []ProofAccount {
	if slices.ContainsFunc(accounts, func(account ProofAccount) bool {
		return account.Address == predeploys.L2ToL1MessagePasserAddr
	}) {
		return accounts
	}
	return append([]ProofAccount{{Address: predeploys.L2ToL1MessagePasserAddr}}, accounts...)
}

// awaitProofsSyncStatus retries debug_proofsSyncStatus until it answers, failing once timeout ends.
func (el *L2ELNode) awaitProofsSyncStatus(timeout time.Duration) ProofsSyncStatus {
	return el.pollProofsSyncStatus(timeout, "answer debug_proofsSyncStatus", func(ProofsSyncStatus) bool { return true })
}

// VerifyProofsMatchAt verifies the proofs of the given accounts at one block the same way
// VerifyHistoricalProofsMatch does, after waiting up to DefaultProofsCatchUpTimeout for the proofs
// store to serve the block. Only the listed accounts are proven, and reference is not checked for
// independence.
func (el *L2ELNode) VerifyProofsMatchAt(reference *L2ELNode, block uint64, accounts ...ProofAccount) {
	window := el.WaitForProofsStoreBlock(block, DefaultProofsCatchUpTimeout)
	el.verifyProofsMatchAt(reference, block, window, accounts, DefaultProofsCatchUpTimeout, false)
}

// VerifyWithdrawalsRootAt requires the header of block to carry, as its withdrawalsRoot, the
// L2ToL1MessagePasser storage root this node's eth_getProof serves at that block, with the proof
// verifying against the header's state root, and returns the header. Only valid from Isthmus on:
// before it the field is the hash of an empty withdrawals list and the check fails.
func (el *L2ELNode) VerifyWithdrawalsRootAt(block uint64) eth.BlockInfo {
	header := el.awaitHeaderByNumber(block, DefaultProofsCatchUpTimeout)
	el.require.NotNilf(header.WithdrawalsRoot(), "block %d on %s carries no withdrawalsRoot", block, el.inner.Name())
	proof, _ := el.verifiedProofAt(header, ProofAccount{Address: predeploys.L2ToL1MessagePasserAddr}, DefaultProofsCatchUpTimeout, false)
	el.requireWithdrawalsRootIsStorageRoot(header, proof.StorageHash)
	el.log.Debug("Withdrawals root is the message passer's storage root", "name", el.inner.Name(), "block", block,
		"withdrawals_root", proof.StorageHash)
	return header
}

// requireWithdrawalsRootIsStorageRoot requires the header's withdrawalsRoot to be storageRoot, the
// L2ToL1MessagePasser storage root this node serves at that block.
func (el *L2ELNode) requireWithdrawalsRootIsStorageRoot(header eth.BlockInfo, storageRoot common.Hash) {
	el.require.Equalf(*header.WithdrawalsRoot(), storageRoot,
		"block %d (%s): the withdrawalsRoot on %s is not the L2ToL1MessagePasser storage root its eth_getProof serves",
		header.NumberU64(), header.Hash(), el.inner.Name())
}

// verifyProofsMatchAt reports whether it skipped the block because the store pruned it while its
// proofs were fetched, which only happens when skipPruned is set.
func (el *L2ELNode) verifyProofsMatchAt(reference *L2ELNode, block uint64, window ProofsSyncStatus, accounts []ProofAccount,
	catchUpTimeout time.Duration, skipPruned bool) (prunedDuringFetch bool) {
	el.require.Truef(window.serves(block), "proofs store on %s does not serve block %d: window %s", el.inner.Name(), block, window)

	header := el.awaitHeaderByNumber(block, catchUpTimeout)
	referenceHeader := reference.awaitHeaderByNumber(block, catchUpTimeout)
	el.require.Equalf(referenceHeader.Hash(), header.Hash(), "block %d: %s and reference %s disagree on the block hash",
		block, el.inner.Name(), reference.inner.Name())

	for _, account := range accounts {
		if el.verifyAccountProofMatchesAt(reference, header, account, catchUpTimeout, skipPruned) {
			el.log.Info("Block left the proofs window while its proofs were fetched; skipping", "name", el.inner.Name(), "block", block)
			return true
		}
	}
	el.log.Info("Historical proofs verified", "name", el.inner.Name(), "block", block, "accounts", len(accounts))
	return false
}

func (el *L2ELNode) verifyAccountProofMatchesAt(reference *L2ELNode, header eth.BlockInfo, account ProofAccount,
	catchUpTimeout time.Duration, skipPruned bool) (prunedDuringFetch bool) {
	block := header.NumberU64()
	proof, pruned := el.verifiedProofAt(header, account, catchUpTimeout, skipPruned)
	if pruned {
		return true
	}
	referenceProof, _ := reference.verifiedProofAt(header, account, catchUpTimeout, false)
	el.require.Equalf(referenceProof, proof, "block %d account %s: eth_getProof from %s differs from reference %s",
		block, account.Address, el.inner.Name(), reference.inner.Name())

	// Isthmus made withdrawalsRoot the message passer's storage root. Before that the header
	// carries the hash of an empty withdrawals list, which says nothing about the message passer.
	if account.Address == predeploys.L2ToL1MessagePasserAddr && header.WithdrawalsRoot() != nil && *header.WithdrawalsRoot() != types.EmptyWithdrawalsHash {
		el.requireWithdrawalsRootIsStorageRoot(header, proof.StorageHash)
	}
	el.log.Debug("Historical proof matches the reference", "name", el.inner.Name(), "block", block,
		"account", account.Address, "storage_keys", len(account.StorageKeys))
	return false
}

// requireIndependentProofReference fails unless reference lacks debug_proofsSyncStatus, which
// op-reth registers only on a proofs-history node and op-geth never does.
func (el *L2ELNode) requireIndependentProofReference(reference *L2ELNode) {
	status, err := reference.ProofsSyncStatus(el.ctx)
	var rpcErr gethrpc.Error
	el.require.Truef(errors.As(err, &rpcErr) && rpcErr.ErrorCode() == int(eth.MethodNotFound),
		"reference %s must serve eth_getProof from its own historical state, not a proofs-history store; debug_proofsSyncStatus returned window %s, err: %v",
		reference.inner.Name(), status, err)
}

// awaitHeaderByNumber retries only while the node does not know the block yet, and fails at once
// on any other error.
func (el *L2ELNode) awaitHeaderByNumber(number uint64, timeout time.Duration) eth.BlockInfo {
	ctx, cancel := context.WithTimeout(el.ctx, timeout)
	defer cancel()
	var header eth.BlockInfo
	var lastErr error
	err := wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
		info, err := el.inner.L2EthClient().InfoByNumber(ctx, number)
		if errors.Is(err, ethereum.NotFound) {
			lastErr = err
			el.log.Debug("Block not known yet; will retry", "name", el.inner.Name(), "block", number)
			return false, nil
		}
		if err != nil {
			lastErr = err
			el.log.Error("Header lookup failed", "name", el.inner.Name(), "block", number, "err", err)
			return false, fmt.Errorf("header lookup rejected without retry: %w", err)
		}
		header = info
		return true, nil
	})
	el.require.NoErrorf(err, "%s did not serve the header of block %d within %s; last error: %v", el.inner.Name(), number, timeout, lastErr)
	return header
}

// verifiedProofAt fetches this node's eth_getProof for account at header, normalized, and requires
// it to verify against the header's state root. It reports pruned, without a proof, when awaitProof
// does.
func (el *L2ELNode) verifiedProofAt(header eth.BlockInfo, account ProofAccount, timeout time.Duration, skipPruned bool) (proof *eth.AccountResult, pruned bool) {
	proof, pruned = el.awaitProof(header, account, timeout, skipPruned)
	if pruned {
		return nil, true
	}
	NormalizeProofResponse(proof)
	el.require.NoErrorf(VerifyAccountProof(proof, header.Root()),
		"block %d (%s) account %s: eth_getProof from %s does not verify against state root %s",
		header.NumberU64(), header.Hash(), account.Address, el.inner.Name(), header.Root())
	return proof, false
}

// awaitProof fetches eth_getProof by block hash, so a block that leaves the canonical chain while
// the sweep runs fails the lookup rather than returning the proof of its replacement. It retries
// only while the node reports no state for the block, and fails at once on any other error. With
// skipPruned set, a failed fetch re-reads the proofs window and stops retrying, reporting pruned,
// once the block has fallen below it.
func (el *L2ELNode) awaitProof(header eth.BlockInfo, account ProofAccount, timeout time.Duration, skipPruned bool) (proof *eth.AccountResult, pruned bool) {
	ctx, cancel := context.WithTimeout(el.ctx, timeout)
	defer cancel()
	// A nil key list encodes as JSON null, which op-reth rejects where op-geth accepts it.
	storageKeys := account.StorageKeys
	if storageKeys == nil {
		storageKeys = []common.Hash{}
	}
	var lastErr error
	err := wait.For(ctx, 200*time.Millisecond, func() (bool, error) {
		result, err := el.inner.L2EthClient().GetProof(ctx, account.Address, storageKeys, header.Hash().String())
		if err != nil {
			lastErr = err
			if skipPruned && el.blockPrunedFromProofsStore(ctx, header.NumberU64()) {
				pruned = true
				return true, nil
			}
			if !isNoStateForBlock(err) {
				el.log.Error("eth_getProof failed", "name", el.inner.Name(), "block", header.NumberU64(),
					"account", account.Address, "err", err)
				return false, fmt.Errorf("eth_getProof rejected without retry: %w", err)
			}
			el.log.Warn("eth_getProof found no state for the block yet; will retry", "name", el.inner.Name(),
				"block", header.NumberU64(), "account", account.Address, "err", err)
			return false, nil
		}
		proof = result
		return true, nil
	})
	el.require.NoErrorf(err, "%s did not serve eth_getProof for account %s at block %d within %s; last error: %v",
		el.inner.Name(), account.Address, header.NumberU64(), timeout, lastErr)
	return proof, pruned
}

// isNoStateForBlock reports whether err is reth's "no state found for block" rejection, which a
// proofs-history store returns for a block outside its window and which clears once the store
// indexes the block. Any other eth_getProof error, such as an unknown or non-canonical block hash
// or a block beyond reth's eth proof window, does not clear by retrying.
func isNoStateForBlock(err error) bool {
	return strings.Contains(err.Error(), "no state found for block")
}

// blockPrunedFromProofsStore reports false when the window cannot be read, so the caller keeps
// treating the block as one the store should serve.
func (el *L2ELNode) blockPrunedFromProofsStore(ctx context.Context, block uint64) bool {
	window, err := el.ProofsSyncStatus(ctx)
	if err != nil {
		el.log.Warn("debug_proofsSyncStatus failed while checking whether a block was pruned", "name", el.inner.Name(),
			"block", block, "err", err)
		return false
	}
	return window.prunedPast(block)
}

// sampleBlockNumbers returns evenly spaced block numbers in [first, last], always including both
// edges: every block when maxSamples is non-positive or covers the range, otherwise
// max(maxSamples, 2) of them.
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
// trie root, which is how reth encodes both; and an empty storage proof list becomes nil.
func NormalizeProofResponse(res *eth.AccountResult) {
	for i := range res.StorageProof {
		if len(res.StorageProof[i].Proof) == 1 && bytes.Equal(res.StorageProof[i].Proof[0], []byte{0x80}) {
			res.StorageProof[i].Proof = []hexutil.Bytes{}
		}
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
	if res.StorageHash == types.EmptyRootHash {
		for i, entry := range res.StorageProof {
			if entry.Value.ToInt().Sign() != 0 {
				return fmt.Errorf("storage value %d at key %s is nonzero but the storage trie is empty", i, entry.Key.String())
			}
		}
	} else {
		for i, entry := range res.StorageProof {
			db, err := proofNodesDB(entry.Proof)
			if err != nil {
				return fmt.Errorf("failed to load the storage proof of storage value %d: %w", i, err)
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

	db, err := proofNodesDB(res.AccountProof)
	if err != nil {
		return fmt.Errorf("failed to load the account proof: %w", err)
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

// proofNodesDB loads RLP-encoded trie nodes into a database keyed the way trie.VerifyProof looks
// them up: by keccak256 of the node, or by the node itself when it is shorter than 32 bytes,
// since small MPT nodes are embedded in their parent rather than hashed.
func proofNodesDB(nodes []hexutil.Bytes) (ethdb.KeyValueReader, error) {
	db := memorydb.New()
	for i, encodedNode := range nodes {
		nodeKey := encodedNode
		if len(encodedNode) >= 32 {
			nodeKey = crypto.Keccak256(encodedNode)
		}
		if err := db.Put(nodeKey, encodedNode); err != nil {
			return nil, fmt.Errorf("failed to load proof node %d into mem db: %w", i, err)
		}
	}
	return db, nil
}
