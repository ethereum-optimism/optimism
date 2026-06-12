package interop

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
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
	// ErrHeadRegression is returned when a result's per-chain L2 head moves
	// backwards relative to the previous entry, or changes hash at an
	// unchanged height. Verified heads are monotone by construction (block
	// numbers are derived arithmetically from timestamps and hashes extend
	// sealed history), so a regression means the observation layer captured
	// chain state that contradicts already-verified history. The verifier
	// treats this as terminal (see isDivergenceError).
	ErrHeadRegression = errors.New("verified head regression: result conflicts with previous entry")
	u64Len            = 8
)

// bucketName is the name of the bbolt bucket used to store verified results.
var bucketName = []byte("verified")

var pendingTransitionBucketName = []byte("pending_transition")
var pendingTransitionKey = []byte("pending")

// PendingInvalidation records a chain invalidation that needs to be executed.
type PendingInvalidation struct {
	ChainID                  eth.ChainID `json:"chainID"`
	BlockID                  eth.BlockID `json:"blockID"`
	Timestamp                uint64      `json:"timestamp"` // the interop decision timestamp
	StateRoot                eth.Bytes32 `json:"stateRoot"`
	MessagePasserStorageRoot eth.Bytes32 `json:"messagePasserStorageRoot"`
}

// VerifiedDB provides persistence for verified timestamps using bbolt.
type VerifiedDB struct {
	db             *bolt.DB
	mu             sync.RWMutex
	firstTimestamp uint64
	lastTimestamp  uint64
	initialized    bool
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

	// Initialize the timestamp bounds from the database
	if err := vdb.initTimestampBounds(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize timestamp bounds: %w", err)
	}

	return vdb, nil
}

// initTimestampBounds scans the database to find the lowest and highest committed timestamp.
func (v *VerifiedDB) initTimestampBounds() error {
	v.firstTimestamp = 0
	v.lastTimestamp = 0
	v.initialized = false
	return v.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil
		}

		c := b.Cursor()
		firstKey, _ := c.First()
		lastKey, _ := c.Last()
		if len(firstKey) == u64Len && len(lastKey) == u64Len {
			v.firstTimestamp = binary.BigEndian.Uint64(firstKey)
			v.lastTimestamp = binary.BigEndian.Uint64(lastKey)
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

	// Check for sequential commitment
	if v.initialized {
		if ts != v.lastTimestamp+1 {
			if ts <= v.lastTimestamp {
				// Idempotent replay: crash recovery may call Commit again after the
				// bbolt write succeeded but before ClearPendingTransition. Compare the
				// deserialized VerifiedResult rather than raw bytes so byte-level
				// drift in encoding/json across Go versions does not turn a legitimate
				// replay into a hard ErrAlreadyCommitted.
				key := timestampToKey(ts)
				var existing VerifiedResult
				err := v.db.View(func(tx *bolt.Tx) error {
					b := tx.Bucket(bucketName)
					val := b.Get(key)
					if val == nil {
						return ErrNotFound
					}
					return json.Unmarshal(val, &existing)
				})
				if err != nil {
					return fmt.Errorf("failed to read existing verified result at %d: %w", ts, err)
				}
				if reflect.DeepEqual(existing, result) {
					return nil
				}
				return fmt.Errorf("%w: %d", ErrAlreadyCommitted, ts)
			}
			return fmt.Errorf("%w: expected %d, got %d", ErrNonSequential, v.lastTimestamp+1, ts)
		}
		// Defense in depth, mirroring the Dafny model's Commit precondition:
		// for chains present in both consecutive entries, the head number
		// must not regress, and an unchanged height must carry an unchanged
		// hash. Chains may join or leave the set between entries (dependency
		// set changes), so only the intersection is compared.
		prev, err := v.getLocked(v.lastTimestamp)
		if err != nil {
			return fmt.Errorf("failed to read previous verified result at %d: %w", v.lastTimestamp, err)
		}
		for chainID, prevHead := range prev.L2Heads {
			newHead, ok := result.L2Heads[chainID]
			if !ok {
				continue
			}
			if newHead.Number < prevHead.Number ||
				(newHead.Number == prevHead.Number && newHead.Hash != prevHead.Hash) {
				return fmt.Errorf("%w: chain %s head %v at ts=%d, previous head %v at ts=%d",
					ErrHeadRegression, chainID, newHead, ts, prevHead, v.lastTimestamp)
			}
		}
	}

	value, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal verified result: %w", err)
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
	if !v.initialized {
		v.firstTimestamp = ts
	}
	v.lastTimestamp = ts
	v.initialized = true

	return nil
}

// Get retrieves the verified result at the given timestamp.
func (v *VerifiedDB) Get(ts uint64) (VerifiedResult, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.getLocked(ts)
}

// getLocked is Get without lock acquisition, for callers already holding v.mu.
func (v *VerifiedDB) getLocked(ts uint64) (VerifiedResult, error) {
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

// FirstTimestamp returns the first committed timestamp.
// Returns 0 and false if no timestamps have been committed.
func (v *VerifiedDB) FirstTimestamp() (uint64, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.firstTimestamp, v.initialized
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

		// Collect the keys to delete first, then delete them. Deleting during
		// Seek/Next iteration is unsafe in bbolt — a delete can shift inodes so
		// Next() skips an entry, leaving keys >= timestamp behind.
		startKey := timestampToKey(timestamp)
		var toDelete [][]byte
		for k, _ := c.Seek(startKey); k != nil; k, _ = c.Next() {
			key := make([]byte, len(k))
			copy(key, k)
			toDelete = append(toDelete, key)
		}
		for _, k := range toDelete {
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

	if !deleted {
		return false, nil
	}

	// Re-derive the bounds in a SEPARATE read transaction, after the delete
	// transaction has committed and the b-tree has rebalanced. Reading
	// First()/Last() inside the mutating transaction is unreliable: after a
	// multi-page tail delete, Cursor.Last() can return nil before commit-time
	// rebalancing, which would wrongly zero the in-memory bounds (reporting the
	// DB as empty while entries remain on disk).
	var nextFirst, nextLast uint64
	var nextInitialized bool
	err = v.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		c := b.Cursor()
		firstKey, _ := c.First()
		lastKey, _ := c.Last()
		if len(firstKey) == u64Len && len(lastKey) == u64Len {
			nextFirst = binary.BigEndian.Uint64(firstKey)
			nextLast = binary.BigEndian.Uint64(lastKey)
			nextInitialized = true
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("failed to read bounds after rewind: %w", err)
	}

	v.firstTimestamp = nextFirst
	v.lastTimestamp = nextLast
	v.initialized = nextInitialized
	return true, nil
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
