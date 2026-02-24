package interop

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

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

// VerifiedDB provides persistence for verified timestamps using bbolt.
type VerifiedDB struct {
	db            *bolt.DB
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

	// Ensure the bucket exists
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
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
	ts := result.Timestamp

	// Check for sequential commitment
	if v.initialized {
		if ts != v.lastTimestamp+1 {
			if ts <= v.lastTimestamp {
				return fmt.Errorf("%w: %d", ErrAlreadyCommitted, ts)
			}
			return fmt.Errorf("%w: expected %d, got %d", ErrNonSequential, v.lastTimestamp+1, ts)
		}
	}

	// Serialize the result
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
	v.lastTimestamp = ts
	v.initialized = true

	return nil
}

// Get retrieves the verified result at the given timestamp.
func (v *VerifiedDB) Get(ts uint64) (VerifiedResult, error) {
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
	return v.lastTimestamp, v.initialized
}

// RewindAfter removes all verified results after the given timestamp.
func (v *VerifiedDB) RewindAfter(timestamp uint64) (bool, error) {
	return v.Rewind(timestamp + 1)
}

// Rewind removes all verified results at or after the given timestamp.
// Returns true if any results were deleted, false otherwise.
func (v *VerifiedDB) Rewind(timestamp uint64) (bool, error) {
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
		// Reinitialize lastTimestamp from the database
		if err := v.initLastTimestamp(); err != nil {
			return deleted, fmt.Errorf("failed to reinitialize lastTimestamp after rewind: %w", err)
		}
		// If no timestamps remain, reset initialized state
		if err := v.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(bucketName)
			c := b.Cursor()
			if k, _ := c.First(); k == nil {
				v.initialized = false
				v.lastTimestamp = 0
			}
			return nil
		}); err != nil {
			return deleted, err
		}
	}

	return deleted, nil
}

// Close closes the database.
func (v *VerifiedDB) Close() error {
	return v.db.Close()
}
