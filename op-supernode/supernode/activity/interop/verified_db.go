package interop

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	verifiedDBName = "VerifiedAtTimestamp"
)

var (
	ErrNotFound         = errors.New("timestamp not found")
	ErrNonSequential    = errors.New("timestamps must be committed sequentially with no gaps")
	ErrAlreadyCommitted = errors.New("timestamp already committed")
)

// VerifiedDB provides persistence for verified timestamps using LevelDB.
type VerifiedDB struct {
	db            *leveldb.DB
	lastTimestamp uint64
	initialized   bool
}

// OpenVerifiedDB opens or creates a VerifiedDB at the given data directory.
func OpenVerifiedDB(dataDir string) (*VerifiedDB, error) {
	dbPath := filepath.Join(dataDir, verifiedDBName)
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb at %s: %w", dbPath, err)
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
	iter := v.db.NewIterator(nil, nil)
	defer iter.Release()

	// Seek to the last key
	if iter.Last() {
		key := iter.Key()
		if len(key) == 8 {
			v.lastTimestamp = binary.BigEndian.Uint64(key)
			v.initialized = true
		}
	}

	return iter.Error()
}

// timestampToKey converts a timestamp to a big-endian byte key.
// Using big-endian ensures lexicographic ordering matches numeric ordering.
func timestampToKey(ts uint64) []byte {
	key := make([]byte, 8)
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
	if err := v.db.Put(key, value, nil); err != nil {
		return fmt.Errorf("failed to write to leveldb: %w", err)
	}

	// Update state
	v.lastTimestamp = ts
	v.initialized = true

	return nil
}

// Get retrieves the verified result at the given timestamp.
func (v *VerifiedDB) Get(ts uint64) (VerifiedResult, error) {
	key := timestampToKey(ts)
	value, err := v.db.Get(key, nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return VerifiedResult{}, ErrNotFound
		}
		return VerifiedResult{}, fmt.Errorf("failed to read from leveldb: %w", err)
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
	return v.db.Has(key, nil)
}

// LastTimestamp returns the most recently committed timestamp.
// Returns 0 and false if no timestamps have been committed.
func (v *VerifiedDB) LastTimestamp() (uint64, bool) {
	return v.lastTimestamp, v.initialized
}

// Close closes the database.
func (v *VerifiedDB) Close() error {
	return v.db.Close()
}
