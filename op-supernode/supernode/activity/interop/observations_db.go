package interop

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	bolt "go.etcd.io/bbolt"
)

const observationsDBName = "ChainObservations"

var (
	observationsBucketName = []byte("snapshot")
	observationsKey        = []byte("current")
)

// ObservationsSnapshot is the per-chain optimistic observation snapshot
// captured by the verifier during a single round, tied to a specific
// timestamp. The verifier replaces it atomically each round so RPC reads see
// a consistent (timestamp, per-chain observations) view tied to the verifier's
// most recent observation pass — not to live SafeDB state which can race with
// chain rewinds.
//
// Only one snapshot is retained at a time. The verifier processes timestamps
// sequentially, so a snapshot covers exactly the next-up timestamp the
// verifier is trying to verify. Once that timestamp is committed as a
// VerifiedRecord, the verified record supersedes the snapshot for queries at
// that timestamp; the snapshot is then overwritten by the next round's
// observation pass at lastVerifiedTS+1.
type ObservationsSnapshot struct {
	Timestamp uint64                           `json:"timestamp"`
	Chains    map[eth.ChainID]ChainObservation `json:"chains"`
}

// ObservationsDB persists ObservationsSnapshot atomically.
type ObservationsDB struct {
	db *bolt.DB
	mu sync.RWMutex
}

func OpenObservationsDB(dataDir string) (*ObservationsDB, error) {
	dbPath := filepath.Join(dataDir, observationsDBName+".db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open observations bbolt at %s: %w", dbPath, err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(observationsBucketName)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create observations bucket: %w", err)
	}
	return &ObservationsDB{db: db}, nil
}

// Set replaces the current snapshot atomically.
func (o *ObservationsDB) Set(snap ObservationsSnapshot) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	value, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshal observations snapshot: %w", err)
	}
	return o.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(observationsBucketName).Put(observationsKey, value)
	})
}

// Get returns the current snapshot, or nil if nothing has been stored.
func (o *ObservationsDB) Get() (*ObservationsSnapshot, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	var snap *ObservationsSnapshot
	err := o.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(observationsBucketName)
		val := b.Get(observationsKey)
		if val == nil {
			return nil
		}
		snap = &ObservationsSnapshot{}
		return json.Unmarshal(val, snap)
	})
	if err != nil {
		return nil, fmt.Errorf("read observations snapshot: %w", err)
	}
	return snap, nil
}

// Clear removes any stored snapshot. Used on rewind so stale per-chain
// observations cannot be served after the verifier's frontier moves
// backward.
func (o *ObservationsDB) Clear() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(observationsBucketName).Delete(observationsKey)
	})
}

// Close closes the database.
func (o *ObservationsDB) Close() error {
	return o.db.Close()
}
