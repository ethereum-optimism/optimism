package interop

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	bolt "go.etcd.io/bbolt"
)

const (
	verifiedDBName = "VerifiedAtTimestamp"
)

var (
	ErrNotFound         = errors.New("timestamp not found")
	ErrNonSequential    = errors.New("timestamps must be committed sequentially with no gaps")
	ErrAlreadyCommitted = errors.New("timestamp already committed")
	u64Len              = 8
)

// bucketName is the name of the bbolt bucket used to store verified results.
var bucketName = []byte("verified")

var pendingTransitionBucketName = []byte("pending_transition")
var pendingTransitionKey = []byte("pending")

// PendingInvalidation records a chain invalidation that needs to be executed.
type PendingInvalidation struct {
	ChainID   eth.ChainID `json:"chainID"`
	BlockID   eth.BlockID `json:"blockID"`
	Timestamp uint64      `json:"timestamp"` // the interop decision timestamp
}

// VerifiedDB provides persistence for verified timestamps using bbolt.
type VerifiedDB struct {
	db            *bolt.DB
	mu            sync.RWMutex
	lastTimestamp uint64
	initialized   bool
}

// OpenVerifiedDB opens or creates a VerifiedDB at the given data directory.
func OpenVerifiedDB(dataDir string) (*VerifiedDB, error) {
	dbPath := filepath.Join(dataDir, verifiedDBName+".db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt at %s: %w", dbPath, err)
	}

	// Ensure the buckets exist
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucketName); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(pendingTransitionBucketName)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	vdb := &VerifiedDB{
		db: db,
	}

	// Initialize the last timestamp from the database
	if err := vdb.initLastTimestamp(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize last timestamp: %w", err)
	}

	return vdb, nil
}

// initLastTimestamp scans the database to find the highest committed timestamp.
func (v *VerifiedDB) initLastTimestamp() error {
	v.lastTimestamp = 0
	v.initialized = false
	return v.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil
		}

		c := b.Cursor()
		key, _ := c.Last()
		if len(key) == u64Len {
			v.lastTimestamp = binary.BigEndian.Uint64(key)
			v.initialized = true
		}

		return nil
	})
}

// timestampToKey converts a timestamp to a big-endian byte key.
// Using big-endian ensures lexicographic ordering matches numeric ordering.
func timestampToKey(ts uint64) []byte {
	key := make([]byte, u64Len)
	binary.BigEndian.PutUint64(key, ts)
	return key
}

// Commit stores a verified result at the given timestamp.
// Timestamps must be committed sequentially with no gaps.
func (v *VerifiedDB) Commit(result VerifiedResult) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	ts := result.Timestamp

	// Serialize the result up front so replay of an already-applied transition can
	// be treated as success when the stored value is identical.
	value, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal verified result: %w", err)
	}

	// Check for sequential commitment
	if v.initialized {
		if ts != v.lastTimestamp+1 {
			if ts <= v.lastTimestamp {
				key := timestampToKey(ts)
				var existing []byte
				err := v.db.View(func(tx *bolt.Tx) error {
					b := tx.Bucket(bucketName)
					val := b.Get(key)
					if val == nil {
						return ErrNotFound
					}
					existing = append(existing[:0], val...)
					return nil
				})
				if err != nil {
					return fmt.Errorf("failed to read existing verified result at %d: %w", ts, err)
				}
				if bytes.Equal(existing, value) {
					return nil
				}
				return fmt.Errorf("%w: %d", ErrAlreadyCommitted, ts)
			}
			return fmt.Errorf("%w: expected %d, got %d", ErrNonSequential, v.lastTimestamp+1, ts)
		}
	}

	// Store in database
	key := timestampToKey(ts)
	err = v.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.Put(key, value)
	})
	if err != nil {
		return fmt.Errorf("failed to write to bbolt: %w", err)
	}

	// Update state
	v.lastTimestamp = ts
	v.initialized = true

	return nil
}

// Get retrieves the verified result at the given timestamp.
func (v *VerifiedDB) Get(ts uint64) (VerifiedResult, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	key := timestampToKey(ts)
	var value []byte

	err := v.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		val := b.Get(key)
		if val == nil {
			return ErrNotFound
		}
		// Copy the value since it's only valid for the life of the transaction
		value = make([]byte, len(val))
		copy(value, val)
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return VerifiedResult{}, ErrNotFound
		}
		return VerifiedResult{}, fmt.Errorf("failed to read from bbolt: %w", err)
	}

	var result VerifiedResult
	if err := json.Unmarshal(value, &result); err != nil {
		return VerifiedResult{}, fmt.Errorf("failed to unmarshal verified result: %w", err)
	}

	return result, nil
}

// Has returns whether a timestamp has been verified.
func (v *VerifiedDB) Has(ts uint64) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	key := timestampToKey(ts)
	var found bool

	err := v.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		found = b.Get(key) != nil
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("failed to check key in bbolt: %w", err)
	}

	return found, nil
}

// LastTimestamp returns the most recently committed timestamp.
// Returns 0 and false if no timestamps have been committed.
func (v *VerifiedDB) LastTimestamp() (uint64, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.lastTimestamp, v.initialized
}

// Rewind removes all verified results at or after the given timestamp.
// Returns true if any results were deleted, false otherwise.
func (v *VerifiedDB) Rewind(timestamp uint64) (bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	var deleted bool

	err := v.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		c := b.Cursor()

		// Start from the timestamp and delete all entries at or after it
		startKey := timestampToKey(timestamp)
		for k, _ := c.Seek(startKey); k != nil; k, _ = c.Next() {
			if err := b.Delete(k); err != nil {
				return err
			}
			deleted = true
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("failed to rewind verifiedDB: %w", err)
	}

	// Update state
	if deleted {
		if err := v.initLastTimestamp(); err != nil {
			return deleted, fmt.Errorf("failed to reinitialize lastTimestamp after rewind: %w", err)
		}
	}

	return deleted, nil
}

// SetPendingTransition persists a generic interop transition as a write-ahead log.
// Must be called BEFORE executing any durable side effects for crash safety.
func (v *VerifiedDB) SetPendingTransition(pending PendingTransition) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	value, err := json.Marshal(pending)
	if err != nil {
		return fmt.Errorf("failed to marshal pending transition: %w", err)
	}
	return v.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(pendingTransitionBucketName)
		return b.Put(pendingTransitionKey, value)
	})
}

// GetPendingTransition retrieves any pending transition from the WAL.
// Returns nil if no pending work exists.
func (v *VerifiedDB) GetPendingTransition() (*PendingTransition, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var pending PendingTransition
	var found bool
	err := v.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(pendingTransitionBucketName)
		val := b.Get(pendingTransitionKey)
		if val == nil {
			return nil
		}
		found = true
		data := make([]byte, len(val))
		copy(data, val)
		return json.Unmarshal(data, &pending)
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &pending, nil
}

// ClearPendingTransition removes the WAL entry after the transition is fully applied.
func (v *VerifiedDB) ClearPendingTransition() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(pendingTransitionBucketName)
		return b.Delete(pendingTransitionKey)
	})
}

// Close closes the database.
func (v *VerifiedDB) Close() error {
	return v.db.Close()
}
