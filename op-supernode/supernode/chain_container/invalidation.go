package chain_container

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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

// DenyRecord stores a denied payload hash along with decision provenance.
type DenyRecord struct {
	PayloadHash       common.Hash `json:"payloadHash"`
	DecisionTimestamp uint64      `json:"decisionTimestamp"`
}

func encodeDenyRecords(records []DenyRecord) ([]byte, error) {
	return json.Marshal(records)
}

func decodeDenyRecords(raw []byte) ([]DenyRecord, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var records []DenyRecord
	if err := json.Unmarshal(raw, &records); err == nil {
		return records, nil
	}
	// Backward compatibility: legacy format is concatenated 32-byte hashes.
	// Legacy entries get DecisionTimestamp: 0, which means they are never
	// removed by PruneAtOrAfterTimestamp (since rewind timestamps are always
	// well above 0). This is the safe default — deny decisions from before
	// provenance tracking was added should be preserved rather than silently
	// dropped.
	if len(raw)%common.HashLength != 0 {
		return nil, fmt.Errorf("invalid denylist record payload length %d", len(raw))
	}
	records = make([]DenyRecord, 0, len(raw)/common.HashLength)
	for i := 0; i+common.HashLength <= len(raw); i += common.HashLength {
		records = append(records, DenyRecord{
			PayloadHash:       common.BytesToHash(raw[i : i+common.HashLength]),
			DecisionTimestamp: 0,
		})
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
// Multiple hashes can be denied at the same height.
func (d *DenyList) Add(height uint64, payloadHash common.Hash, decisionTimestamp uint64) error {
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

		// Check if hash already exists
		for _, r := range records {
			if r.PayloadHash == payloadHash {
				return nil
			}
		}

		records = append(records, DenyRecord{
			PayloadHash:       payloadHash,
			DecisionTimestamp: decisionTimestamp,
		})

		encoded, err := encodeDenyRecords(records)
		if err != nil {
			return err
		}
		return b.Put(key, encoded)
	})
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

// InvalidateBlock adds a block to the deny list and triggers a rewind if the chain
// currently uses that block at the specified height.
// WARNING: this should only be called by interop transition application.
// Other callers risk triggering chain rewinds outside the interop WAL model.
// TODO(#19561): remove this footgun by moving reorg-triggering operations behind a
// smaller interop-owned interface.
// Returns true if a rewind was triggered, false otherwise.
// Note: Genesis block (height=0) cannot be invalidated as there is no prior block to rewind to.
func (c *simpleChainContainer) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64) (bool, error) {
	if c.denyList == nil {
		return false, fmt.Errorf("deny list not initialized")
	}

	// Cannot invalidate genesis block - there is no prior block to rewind to
	if height == 0 {
		return false, fmt.Errorf("cannot invalidate genesis block (height=0)")
	}

	// Add to deny list first
	if err := c.denyList.Add(height, payloadHash, decisionTimestamp); err != nil {
		return false, fmt.Errorf("failed to add block to deny list: %w", err)
	}

	c.log.Info("added block to deny list",
		"height", height,
		"payloadHash", payloadHash,
	)

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
	priorTimestamp := c.blockNumberToTimestamp(height - 1)
	if err := c.RewindEngine(ctx, priorTimestamp, invalidatedBlock); err != nil {
		return false, fmt.Errorf("failed to rewind engine: %w", err)
	}

	c.log.Info("rewind completed after block invalidation",
		"invalidatedHeight", height,
		"rewindToTimestamp", priorTimestamp,
	)

	return true, nil
}

func (c *simpleChainContainer) PruneDeniedAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error) {
	if c.denyList == nil {
		return nil, fmt.Errorf("deny list not initialized")
	}
	return c.denyList.PruneAtOrAfterTimestamp(timestamp)
}
