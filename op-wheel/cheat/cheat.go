package cheat

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var HundredETH = big.NewInt(0).Mul(big.NewInt(100), big.NewInt(1000000000000000000))

type Cheater struct {
	// The database of the chain with the head block that we patch the state-root of, once the state is updated.
	DB ethdb.Database
	// Initialized chain, wrapping the database with in-memory presentation of headers and recent changes and such.
	Blockchain *core.BlockChain
	// The Cheater avoids making writes if this is set to True, and opens the DB as readonly.
	ReadOnly bool
}

func OpenGethRawDB(dataDirPath string, readOnly bool) (ethdb.Database, error) {
	// don't use readonly mode in actual DB, it doesn't work with Geth.
	db, err := rawdb.Open(rawdb.OpenOptions{
		Type:              "leveldb",
		Directory:         dataDirPath,
		AncientsDirectory: filepath.Join(dataDirPath, "ancient"),
		Namespace:         "",
		Cache:             2048,
		Handles:           500,
		ReadOnly:          readOnly,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb: %w", err)
	}
	return db, nil
}

// OpenGethDB opens a geth database to apply cheats to.
func OpenGethDB(dataDirPath string, readOnly bool) (*Cheater, error) {
	db, err := OpenGethRawDB(dataDirPath, readOnly)
	if err != nil {
		return nil, err
	}
	ch, err := core.NewBlockChain(db, nil, nil, nil,
		beacon.New(ethash.NewFullFaker()), vm.Config{}, nil, nil)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to open blockchain around chain db: %w", err)
	}
	return &Cheater{
		DB:         db,
		Blockchain: ch,
		ReadOnly:   readOnly,
	}, nil
}

func (ch *Cheater) Close() error {
	return ch.DB.Close()
}

type HeadFn func(headState *state.StateDB) error

// RunAndClose runs the given function on the head-state, and then persists any changes (if not ReadOnly),
// and updates the blockchain headers indexes to reflect the new state-root, so geth will believe the cheat
// (unless it ever re-applies the block).
func (ch *Cheater) RunAndClose(fn HeadFn) error {
	preBlock := ch.Blockchain.CurrentBlock()
	if a, b := preBlock.NumberU64(), ch.Blockchain.Genesis().NumberU64(); a <= b {
		return fmt.Errorf("cheating at genesis (head block %d <= genesis block %d) is not supported", a, b)
	}
	state, err := ch.Blockchain.StateAt(preBlock.Root())
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("failed to look up head state: %w", err)
	}
	if err := fn(state); err != nil {
		_ = ch.Close()
		return fmt.Errorf("failed to run state change: %w", err)
	}
	if ch.ReadOnly {
		return ch.Close()
	}

	// commit the changes, and then update the state-root
	stateRoot, err := state.Commit(true)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("failed to commit state change: %w", err)
	}
	header := preBlock.Header()
	header.Root = stateRoot
	blockHash := header.Hash()

	// We have to manually commit the updated state root to the database.
	if err := state.Database().TrieDB().Commit(stateRoot, true); err != nil {
		return fmt.Errorf("error committing trie db: %w", err)
	}

	// based on core.BlockChain.writeHeadBlock:
	// Add the block to the canonical chain number scheme and mark as the head
	batch := ch.DB.NewBatch()
	if ch.Blockchain.CurrentFinalizedBlock().Hash() == preBlock.Hash() {
		rawdb.WriteFinalizedBlockHash(batch, blockHash)
	}
	rawdb.DeleteHeaderNumber(batch, preBlock.Hash())
	rawdb.WriteHeadHeaderHash(batch, blockHash)
	rawdb.WriteHeadFastBlockHash(batch, blockHash)
	rawdb.WriteCanonicalHash(batch, blockHash, preBlock.NumberU64())
	rawdb.WriteHeaderNumber(batch, blockHash, preBlock.NumberU64())
	rawdb.WriteHeader(batch, header)
	// not keyed by blockhash, and we didn't remove any txs, so we just leave this one as-is.
	// rawdb.WriteTxLookupEntriesByBlock(batch, block)
	rawdb.WriteHeadBlockHash(batch, blockHash)

	// Geth stores the TD for each block separately from the block itself. We must update this
	// manually, otherwise Geth thinks we haven't reached TTD yet and tries to build a block
	// using Clique consensus, which causes a panic.
	rawdb.WriteTd(batch, blockHash, preBlock.NumberU64(), ch.Blockchain.GetTd(preBlock.Hash(), preBlock.NumberU64()))

	// Need to copy over receipts since they are keyed by block hash.
	receipts := rawdb.ReadReceipts(ch.DB, preBlock.Hash(), preBlock.NumberU64(), ch.Blockchain.Config())
	rawdb.WriteReceipts(batch, blockHash, preBlock.NumberU64(), receipts)

	// Geth maintains an internal mapping between block bodies and their hashes. None of the database
	// accessors above update this mapping, so we need to do it manually.
	oldKey := blockBodyKey(preBlock.NumberU64(), preBlock.Hash())
	oldBody := rawdb.ReadBodyRLP(ch.DB, preBlock.Hash(), preBlock.NumberU64())
	newKey := blockBodyKey(preBlock.NumberU64(), blockHash)
	if err := batch.Delete(oldKey); err != nil {
		return fmt.Errorf("error deleting old block body key")
	}
	if err := batch.Put(newKey, oldBody); err != nil {
		return fmt.Errorf("error setting new block body key")
	}

	// Flush the whole batch into the disk, exit the node if failed
	if err := batch.Write(); err != nil {
		_ = ch.Close()
		return fmt.Errorf("failed to update chain indexes and markers: %w", err)
	}
	// Technically there are more in-memory things to update in real geth,
	// to which we don't even have public API access, but that's fine, we're done.
	// *And we did update the finalized marker, which is flushed from memory to disk on shutdown in geth.
	// bc.hc.SetCurrentHeader(block.Header())
	// headFastBlockGauge.Update(int64(block.NumberU64()))
	// headBlockGauge.Update(int64(block.NumberU64()))

	return ch.Close()
}

// StorageSet modifies the storage of the given address at the given key to the given value.
func StorageSet(address common.Address, key common.Hash, value common.Hash) HeadFn {
	return func(headState *state.StateDB) error {
		headState.SetState(address, key, value)
		return nil
	}
}

// StorageGet just reads the storage of the given address at the given key.
func StorageGet(address common.Address, key common.Hash, w io.Writer) HeadFn {
	return func(headState *state.StateDB) error {
		value := headState.GetState(address, key)
		_, err := io.WriteString(w, value.Hex())
		return err
	}
}

// StorageReadAll reads all values of the given address, and writes it as a (+) diff to the given output writer.
// Simply replace the (+) with (-) if you need to apply the diff as removal of values.
// Combined with StoragePatch this allows for quick surgery of 1 account in one database,
// to another account (maybe even in a different database!).
func StorageReadAll(address common.Address, w io.Writer) HeadFn {
	return func(headState *state.StateDB) error {
		storage, err := headState.StorageTrie(address)
		if err != nil {
			return fmt.Errorf("failed to open storage trie of addr %s: %w", address, err)
		}
		if storage == nil {
			return fmt.Errorf("no storage trie in state for account %s", address)
		}
		iter := trie.NewIterator(storage.NodeIterator(nil))
		for iter.Next() {
			if _, err := fmt.Fprintf(w, "+ %x = %x\n", iter.Key, dbValueToHash(iter.Value)); err != nil {
				return err
			}
		}
		return nil
	}
}

func dbValueToHash(enc []byte) common.Hash {
	var value common.Hash
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			panic(err)
		}
		value.SetBytes(content)
	}
	return value
}

// StorageDiff compares the storage of two different accounts, and writes a patch with differences.
// Each difference is expressed with 1 character + or - to indicate the change from a to b, followed by key = value.
func StorageDiff(out io.Writer, addressA, addressB common.Address) HeadFn {
	return func(headState *state.StateDB) error {
		aStorage, err := headState.StorageTrie(addressA)
		if err != nil {
			return fmt.Errorf("failed to open storage trie of addr A %s: %w", addressA, err)
		}
		if aStorage == nil {
			return fmt.Errorf("no storage trie in state for account A %s", addressA)
		}
		bStorage, err := headState.StorageTrie(addressB)
		if err != nil {
			return fmt.Errorf("failed to open storage trie of addr B %s: %w", addressB, err)
		}
		if bStorage == nil {
			return fmt.Errorf("no storage trie in state for account B %s", addressB)
		}
		aIter := trie.NewIterator(aStorage.NodeIterator(nil))
		bIter := trie.NewIterator(bStorage.NodeIterator(nil))
		hasA := aIter.Next()
		hasB := bIter.Next()
		for {
			if !hasA && !hasB {
				break
			}
			if cmp := bytes.Compare(aIter.Key, bIter.Key); cmp < 0 {
				// a is smaller, and thus missing in b. Print and move forward a
				if _, err := fmt.Fprintf(out, "- %x = %x\n", aIter.Key, dbValueToHash(aIter.Value)); err != nil {
					return err
				}
				hasA = aIter.Next()
			} else if cmp > 0 {
				// b is smaller, and thus missing in a. Print and move forward b
				if _, err := fmt.Fprintf(out, "+ %x = %x\n", bIter.Key, dbValueToHash(bIter.Value)); err != nil {
					return err
				}
				hasB = bIter.Next()
			} else if cmp == 0 {
				// same key, now check if the values differ
				if !bytes.Equal(aIter.Value, bIter.Value) {
					if _, err := fmt.Fprintf(out, "- %x = %x\n", aIter.Key, dbValueToHash(aIter.Value)); err != nil {
						return err
					}
					if _, err := fmt.Fprintf(out, "+ %x = %x\n", bIter.Key, dbValueToHash(bIter.Value)); err != nil {
						return err
					}
				}
				// move both
				hasA = aIter.Next()
				hasB = bIter.Next()
			}
		}
		return nil
	}
}

// StoragePatch applies a patch of changes to the given state account trie.
// Changes are hex encoded key-value pairs separated by (=).
// Additions are prefixed with (+).
// Deletions are prefixed with (-) and overwrite it to a zero value.
// Comments (#) and empty lines are ignored.
func StoragePatch(patch io.Reader, address common.Address) HeadFn {
	return func(headState *state.StateDB) error {
		s := bufio.NewScanner(patch)
		i := 0
		for s.Scan() {
			line := s.Text()
			if len(line) < 1 || line[0] == '#' { // skip empty lines and comments
				continue
			}
			parts := strings.Split(line[1:], "=")
			keyHex := strings.TrimSpace(parts[0])
			valueHex := strings.TrimSpace(parts[1])
			var key, value common.Hash
			if err := key.UnmarshalText([]byte(keyHex)); err != nil {
				return fmt.Errorf("key %s is malformatted: %w", keyHex, err)
			}
			if err := value.UnmarshalText([]byte(valueHex)); err != nil {
				return fmt.Errorf("key %s has malformatted value %s: %w", keyHex, valueHex, err)
			}
			switch line[0] {
			case '+':
				headState.SetState(address, key, value)
			case '-':
				headState.SetState(address, key, common.Hash{})
			default:
				return fmt.Errorf("unrecognized line diff token")
			}
			i += 1
			if i%1000 == 0 { // for every 1000 values, commit to disk
				if _, err := headState.Commit(true); err != nil {
					return fmt.Errorf("failed to commit state to disk after patching %d entries: %w", i, err)
				}
			}
		}
		return nil
	}
}

type OvmOwnersConfig struct {
	Owner     common.Address `json:"owner"`
	Sequencer common.Address `json:"sequencer"`
	Proposer  common.Address `json:"proposer"`
}

func OvmOwners(conf *OvmOwnersConfig) HeadFn {
	return func(headState *state.StateDB) error {
		// Address manager owner
		headState.SetState(common.HexToAddress("0xa6f73589243a6A7a9023b1Fa0651b1d89c177111"), common.Hash{}, conf.Owner.Hash())
		// L1SB proxy owner
		headState.SetState(common.HexToAddress("0x636Af16bf2f682dD3109e60102b8E1A089FedAa8"), common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103"), conf.Owner.Hash())
		// L1XDM owner
		headState.SetState(common.HexToAddress("0x5086d1eEF304eb5284A0f6720f79403b4e9bE294"), common.Hash{31: 0x33}, conf.Owner.Hash())
		// L1 ERC721 bridge owner
		headState.SetState(common.HexToAddress("0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9"), common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103"), conf.Owner.Hash())
		// Legacy sequencer/proposer addresses
		headState.SetState(common.HexToAddress("0xa6f73589243a6A7a9023b1Fa0651b1d89c177111"), common.HexToHash("0x2e0dfce60e9e27f035ce28f63c1bdd77cff6b13d8909da4d81d623ff9123fbdc"), conf.Sequencer.Hash())
		headState.SetState(common.HexToAddress("0xa6f73589243a6A7a9023b1Fa0651b1d89c177111"), common.HexToHash("0x9776dbdebd0d5eedaea450b21da9901ecd5254e5136a3a9b7b0ecd532734d5b5"), conf.Proposer.Hash())
		// Fund sequencer and proposer with 100 ETH
		headState.SetBalance(conf.Sequencer, HundredETH)
		headState.SetBalance(conf.Proposer, HundredETH)
		return nil
	}
}

func SetBalance(addr common.Address, amount *big.Int) HeadFn {
	return func(headState *state.StateDB) error {
		headState.SetBalance(addr, amount)
		return nil
	}
}

func SetNonce(addr common.Address, nonce uint64) HeadFn {
	return func(headState *state.StateDB) error {
		headState.SetNonce(addr, nonce)
		return nil
	}
}

// blockBodyKey returns the database key to use for storing the body of a block.
// This function was copied from Geth's core/rawdb/accessors_chain.go.
func blockBodyKey(number uint64, hash common.Hash) []byte {
	return append(append([]byte("b"), encodeBlockNumber(number)...), hash.Bytes()...)
}

// encodeBlockNumber encodes a block number as big endian uint64. This function was
// copied from Geth's core/rawdb/accessors_chain.go file.
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}
