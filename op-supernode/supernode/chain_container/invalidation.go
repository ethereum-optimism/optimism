package chain_container

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	bolt "go.etcd.io/bbolt"
)

const (
	denyListDBName = "denylist"
)

// denyListBucketName is the name of the bbolt bucket used to store denied block hashes.
var denyListBucketName = []byte("denied_blocks")

// DenyList provides persistence for invalid block payload hashes using bbolt.
// Blocks are keyed by block height, with each height potentially having multiple denied hashes.
type DenyList struct {
	db *bolt.DB
	mu sync.RWMutex
}

// DenyRecord stores a denied payload hash along with decision provenance
// and the output preimage fields for optimistic root computation.
type DenyRecord struct {
	PayloadHash              common.Hash `json:"payloadHash"`
	DecisionTimestamp        uint64      `json:"decisionTimestamp"`
	StateRoot                eth.Bytes32 `json:"stateRoot"`
	MessagePasserStorageRoot eth.Bytes32 `json:"messagePasserStorageRoot"`
}

func encodeDenyRecords(records []DenyRecord) ([]byte, error) {
	return json.Marshal(records)
}

func decodeDenyRecords(raw []byte) ([]DenyRecord, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var records []DenyRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, fmt.Errorf("failed to decode denylist records: %w", err)
	}
	return records, nil
}

// OpenDenyList opens or creates a DenyList at the given data directory.
func OpenDenyList(dataDir string) (*DenyList, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create denylist directory %s: %w", dataDir, err)
	}
	dbPath := filepath.Join(dataDir, denyListDBName+".db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open denylist bbolt at %s: %w", dbPath, err)
	}

	// Ensure the bucket exists
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(denyListBucketName)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create denylist bucket: %w", err)
	}

	return &DenyList{db: db}, nil
}

// heightToKey converts a block height to a big-endian byte key.
// Using big-endian ensures lexicographic ordering matches numeric ordering.
func heightToKey(height uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, height)
	return key
}

// Add adds a payload hash to the deny list at the given block height.
// stateRoot and messagePasserStorageRoot are the output preimage fields for optimistic root computation.
// Multiple hashes can be denied at the same height.
func (d *DenyList) Add(height uint64, payloadHash common.Hash, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := heightToKey(height)

	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(denyListBucketName)

		existing := b.Get(key)
		records, err := decodeDenyRecords(existing)
		if err != nil {
			return err
		}

		for _, r := range records {
			if r.PayloadHash == payloadHash {
				return nil
			}
		}

		records = append(records, DenyRecord{
			PayloadHash:              payloadHash,
			DecisionTimestamp:        decisionTimestamp,
			StateRoot:                stateRoot,
			MessagePasserStorageRoot: messagePasserStorageRoot,
		})

		encoded, err := encodeDenyRecords(records)
		if err != nil {
			return err
		}
		return b.Put(key, encoded)
	})
}

// LastDeniedOutputV0 returns the OutputV0 for the most recently denied block at the given height.
// Returns nil if no blocks are denied at that height.
// Note: supernode does not currently behave in well defined ways when there are multiple denied blocks at the same height.
func (d *DenyList) LastDeniedOutputV0(height uint64) (*eth.OutputV0, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := heightToKey(height)
	var result *eth.OutputV0

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(denyListBucketName)
		existing := b.Get(key)
		if existing == nil {
			return nil
		}

		records, err := decodeDenyRecords(existing)
		if err != nil {
			return err
		}
		if len(records) > 0 {
			r := records[len(records)-1]
			result = &eth.OutputV0{
				StateRoot:                r.StateRoot,
				MessagePasserStorageRoot: r.MessagePasserStorageRoot,
				BlockHash:                r.PayloadHash,
			}
		}
		return nil
	})

	return result, err
}

// GetOutputV0 reconstructs and returns the full OutputV0 for a denied block.
// Returns nil if the hash is not denied at that height.
func (d *DenyList) GetOutputV0(height uint64, payloadHash common.Hash) (*eth.OutputV0, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := heightToKey(height)
	var result *eth.OutputV0

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(denyListBucketName)
		existing := b.Get(key)
		if existing == nil {
			return nil
		}

		records, err := decodeDenyRecords(existing)
		if err != nil {
			return err
		}
		for _, r := range records {
			if r.PayloadHash == payloadHash {
				result = &eth.OutputV0{
					StateRoot:                r.StateRoot,
					MessagePasserStorageRoot: r.MessagePasserStorageRoot,
					BlockHash:                payloadHash,
				}
				return nil
			}
		}
		return nil
	})

	return result, err
}

// Contains checks if a payload hash is denied at the given block height.
func (d *DenyList) Contains(height uint64, payloadHash common.Hash) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := heightToKey(height)
	var found bool

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(denyListBucketName)
		existing := b.Get(key)
		if existing == nil {
			return nil
		}

		records, err := decodeDenyRecords(existing)
		if err != nil {
			return err
		}
		for _, r := range records {
			if r.PayloadHash == payloadHash {
				found = true
				return nil
			}
		}
		return nil
	})

	return found, err
}

// GetDeniedHashes returns all denied payload hashes at the given block height.
func (d *DenyList) GetDeniedHashes(height uint64) ([]common.Hash, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := heightToKey(height)
	var hashes []common.Hash

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(denyListBucketName)
		existing := b.Get(key)
		if existing == nil {
			return nil
		}

		records, err := decodeDenyRecords(existing)
		if err != nil {
			return err
		}
		for _, r := range records {
			hashes = append(hashes, r.PayloadHash)
		}
		return nil
	})

	return hashes, err
}

// GetDeniedRecords returns all denied records at the given block height.
func (d *DenyList) GetDeniedRecords(height uint64) ([]DenyRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := heightToKey(height)
	var records []DenyRecord

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(denyListBucketName)
		existing := b.Get(key)
		if existing == nil {
			return nil
		}

		var decErr error
		records, decErr = decodeDenyRecords(existing)
		return decErr
	})

	return records, err
}

// HasDeniedAtOrAfterTimestamp returns true if any denied payload has
// DecisionTimestamp >= timestamp.
func (d *DenyList) HasDeniedAtOrAfterTimestamp(timestamp uint64) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var found bool
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(denyListBucketName)
		c := b.Cursor()

		for _, v := c.First(); v != nil; _, v = c.Next() {
			records, err := decodeDenyRecords(v)
			if err != nil {
				return err
			}
			for _, r := range records {
				if r.DecisionTimestamp >= timestamp {
					found = true
					return nil
				}
			}
		}
		return nil
	})
	return found, err
}

// PruneAtOrAfterTimestamp iterates all keys in the bucket, decodes records,
// removes any where DecisionTimestamp >= timestamp, re-encodes remaining.
// Returns map of removed hashes by height.
func (d *DenyList) PruneAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	removed := make(map[uint64][]common.Hash)

	err := d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(denyListBucketName)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			height := binary.BigEndian.Uint64(k)

			records, err := decodeDenyRecords(v)
			if err != nil {
				return err
			}

			var kept []DenyRecord
			for _, r := range records {
				if r.DecisionTimestamp >= timestamp {
					removed[height] = append(removed[height], r.PayloadHash)
				} else {
					kept = append(kept, r)
				}
			}

			if len(kept) == 0 {
				if err := b.Delete(k); err != nil {
					return err
				}
			} else if len(kept) < len(records) {
				encoded, err := encodeDenyRecords(kept)
				if err != nil {
					return err
				}
				if err := b.Put(k, encoded); err != nil {
					return err
				}
			}
		}
		return nil
	})

	return removed, err
}

// Close closes the database.
func (d *DenyList) Close() error {
	return d.db.Close()
}

// InvalidateBlock is part of the InteropChain interface — callers must hold
// that wider interface (only interop transition application does) to invoke it.
// Adds a block to the deny list and triggers a rewind if the chain currently
// uses that block at the specified height.
// Returns true if a rewind was triggered, false otherwise.
// Note: Genesis block (height=0) cannot be invalidated as there is no prior block to rewind to.
func (c *simpleChainContainer) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32) (bool, error) {
	if c.denyList == nil {
		return false, fmt.Errorf("deny list not initialized")
	}

	// Cannot invalidate genesis block - there is no prior block to rewind to
	if height == 0 {
		return false, fmt.Errorf("cannot invalidate genesis block (height=0)")
	}

	// Add to deny list with the output preimage fields
	if err := c.denyList.Add(height, payloadHash, decisionTimestamp, stateRoot, messagePasserStorageRoot); err != nil {
		return false, fmt.Errorf("failed to add block to deny list: %w", err)
	}

	c.log.Info("added block to deny list",
		"height", height,
		"payloadHash", payloadHash,
	)

	if c.metrics != nil {
		c.metrics.DenyListEntries.WithLabelValues(c.chainID.String()).Inc()
	}

	// Check if the current chain uses this block at this height
	if c.engine == nil {
		c.log.Warn("engine not initialized, cannot check current block")
		return false, nil
	}

	currentBlock, err := c.engine.L2BlockRefByNumber(ctx, height)
	if err != nil {
		c.log.Warn("failed to get current block at height", "height", height, "err", err)
		return false, nil
	}

	// Compare the current block hash with the invalidated hash
	if currentBlock.Hash != payloadHash {
		c.log.Info("current block differs from invalidated block, no rewind needed",
			"height", height,
			"currentHash", currentBlock.Hash,
			"invalidatedHash", payloadHash,
		)
		return false, nil
	}

	c.log.Warn("current block matches invalidated block, initiating rewind",
		"height", height,
		"hash", payloadHash,
	)

	invalidatedBlock := currentBlock.BlockRef()

	// Rewind to the prior block's timestamp
	priorTimestamp, err := c.BlockNumberToTimestamp(ctx, height-1)
	if err != nil {
		return false, fmt.Errorf("failed to compute rewind timestamp: %w", err)
	}
	if err := c.RewindEngine(ctx, priorTimestamp, invalidatedBlock); err != nil {
		return false, fmt.Errorf("failed to rewind engine: %w", err)
	}

	c.log.Info("rewind completed after block invalidation",
		"invalidatedHeight", height,
		"rewindToTimestamp", priorTimestamp,
	)

	// Record rewind depth: invalidated block was at `height`, rewound to height-1.
	if c.metrics != nil {
		c.metrics.ChainRewindDepthBlocks.WithLabelValues(c.chainID.String()).Observe(1)
	}

	return true, nil
}

func (c *simpleChainContainer) PruneDeniedAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error) {
	if c.denyList == nil {
		return nil, fmt.Errorf("deny list not initialized")
	}
	return c.denyList.PruneAtOrAfterTimestamp(timestamp)
}

func (c *simpleChainContainer) HasDeniedAtOrAfterTimestamp(timestamp uint64) (bool, error) {
	if c.denyList == nil {
		return false, fmt.Errorf("deny list not initialized")
	}
	return c.denyList.HasDeniedAtOrAfterTimestamp(timestamp)
}
