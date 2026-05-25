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

// CanonicalDeniedHeight walks denied heights from highest to lowest and returns
// the lowest height whose denied payload hash matches the canonical hash at
// that height (as reported by `canonical`). Iteration stops at the first height
// where no denied entry is canonical, assuming canonical-vs-denied status is
// monotone in height (lower heights are parents of higher ones, so once we hit
// a non-canonical entry while walking down, lower entries are expected to have
// already been reorged out by an earlier remediation cycle).
//
// Returns (height, true, nil) when a canonical denied entry is found.
// Returns (0, false, nil) when the deny list is empty or no entries are canonical.
//
// `canonical` returns the canonical block hash at the given height. If it
// returns an error (e.g. the EL hasn't synced that far yet), that height is
// treated as non-canonical and iteration stops.
func (d *DenyList) CanonicalDeniedHeight(ctx context.Context, canonical func(ctx context.Context, height uint64) (common.Hash, error)) (uint64, bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var (
		lowestCanonical uint64
		found           bool
	)

	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(denyListBucketName)
		c := b.Cursor()

		// Reverse iteration: highest height first.
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			if err := ctx.Err(); err != nil {
				return err
			}
			height := binary.BigEndian.Uint64(k)

			records, decErr := decodeDenyRecords(v)
			if decErr != nil {
				return decErr
			}
			if len(records) == 0 {
				continue
			}

			canonHash, cErr := canonical(ctx, height)
			if cErr != nil {
				// Canonical probe failed — treat as non-canonical and stop.
				return nil
			}

			var match bool
			for _, r := range records {
				if r.PayloadHash == canonHash {
					match = true
					break
				}
			}
			if !match {
				// First non-canonical entry — stop iteration.
				return nil
			}
			lowestCanonical = height
			found = true
		}
		return nil
	})

	if err != nil {
		return 0, false, err
	}
	return lowestCanonical, found, nil
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

// RecordInvalidation is part of the InteropChain interface — callers must hold
// that wider interface (only interop transition application does) to invoke it.
//
// Persists a denied payload hash (plus its OutputV0 preimages) at the given
// L2 block height. The actual chain reorg is driven separately: when the
// virtual node restarts, op-node's reset path consults
// SuperAuthority.CanonicalDeniedHeight and caps the safe head to (height - 1).
// Derivation then re-derives the denied block, sees it on the deny list via
// payload_process.go IsDenied, and emits deposits-only attributes that replace
// the block via consolidation; the unsafe chain is reorged by op-reth on the
// resulting FCU.
//
// Genesis block (height=0) cannot be invalidated as there is no prior block to
// cap to. StateRoot and MessagePasserStorageRoot must be non-zero: downstream
// optimistic-output-root computation depends on them, and zero values would
// silently produce incorrect output roots.
func (c *simpleChainContainer) RecordInvalidation(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64, stateRoot, messagePasserStorageRoot eth.Bytes32) error {
	if c.denyList == nil {
		return fmt.Errorf("deny list not initialized")
	}
	if height == 0 {
		return fmt.Errorf("cannot invalidate genesis block (height=0)")
	}
	var zero eth.Bytes32
	if stateRoot == zero {
		return fmt.Errorf("refusing to record invalidation with zero state root at height %d", height)
	}
	if messagePasserStorageRoot == zero {
		return fmt.Errorf("refusing to record invalidation with zero message-passer storage root at height %d", height)
	}

	if err := c.denyList.Add(height, payloadHash, decisionTimestamp, stateRoot, messagePasserStorageRoot); err != nil {
		return fmt.Errorf("failed to add block to deny list: %w", err)
	}

	c.log.Info("recorded invalidation in deny list",
		"height", height,
		"payloadHash", payloadHash,
	)

	if c.metrics != nil {
		c.metrics.DenyListEntries.WithLabelValues(c.chainID.String()).Inc()
	}

	return nil
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
