package silhouette

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"

	bolt "go.etcd.io/bbolt"
)

const (
	persistenceVersion = 1
	persistenceDBName  = "silhouette-el.db"
)

var (
	persistenceBucket = []byte("silhouette-el")
	persistenceKey    = []byte("state")
)

// persistentStore is deliberately private to FactStore. The standalone EL is the only owner of
// this database; the supernode consumes its public view over RPC and has no duplicate chain state.
type persistentStore struct {
	db *bolt.DB
}

type trackerState struct {
	Initialized bool                   `json:"initialized"`
	Start       uint64                 `json:"start"`
	Next        uint64                 `json:"next"`
	Processed   map[uint64]common.Hash `json:"processed"`
}

type persistedFact struct {
	Hash common.Hash `json:"hash"`
	Fact Fact        `json:"fact"`
}

type persistedRendering struct {
	Hash      common.Hash `json:"hash"`
	Rendering Rendering   `json:"rendering"`
}

type persistedDenial struct {
	Hash   common.Hash `json:"hash"`
	Number uint64      `json:"number"`
}

type persistedDeniedFact struct {
	Number uint64 `json:"number"`
	Fact   Fact   `json:"fact"`
}

type persistedState struct {
	Version      uint64                `json:"version"`
	Blocks       []Fact                `json:"blocks"`
	Carriers     []carrier             `json:"carriers"`
	Denied       []persistedDenial     `json:"denied"`
	DeniedFacts  []persistedDeniedFact `json:"deniedFacts"`
	Renderings   []persistedRendering  `json:"renderings"`
	Cursors      Cursors               `json:"cursors"`
	Replacements []Fact                `json:"replacements"`
	RewindFacts  []Fact                `json:"rewindFacts"`
	Tracker      trackerState          `json:"tracker"`
}

// OpenFactStore opens or creates the complete durable state of a standalone Silhouette EL.
// bbolt transactions make every Flush an fsync-backed atomic checkpoint: after a crash, callers
// observe either the previous complete state or the next complete state, never a mixture.
func OpenFactStore(dataDir string) (*FactStore, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("silhouette EL data directory is required")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create silhouette EL data directory %q: %w", dataDir, err)
	}
	dbPath := filepath.Join(dataDir, persistenceDBName)
	db, err := bolt.Open(dbPath, 0o600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("open silhouette EL database %q: %w", dbPath, err)
	}
	store := &FactStore{persist: &persistentStore{db: db}}
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(persistenceBucket)
		return err
	}); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize silhouette EL database: %w", err)
	}
	if err := store.restore(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (f *FactStore) restore() error {
	var raw []byte
	if err := f.persist.db.View(func(tx *bolt.Tx) error {
		value := tx.Bucket(persistenceBucket).Get(persistenceKey)
		if value != nil {
			raw = append([]byte(nil), value...)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("read silhouette EL state: %w", err)
	}
	if len(raw) == 0 {
		return nil
	}
	var state persistedState
	if err := json.Unmarshal(raw, &state); err != nil {
		return fmt.Errorf("decode silhouette EL state: %w", err)
	}
	if state.Version != persistenceVersion {
		return fmt.Errorf("unsupported silhouette EL state version %d (want %d)", state.Version, persistenceVersion)
	}
	f.blocks = state.Blocks
	f.carriers = state.Carriers
	f.cursors = state.Cursors
	f.tracker = cloneTrackerState(state.Tracker)
	f.byHash = make(map[common.Hash]uint64, len(f.blocks))
	for _, fact := range f.blocks {
		f.byHash[fact.Hash] = fact.Number
	}
	f.denied = make(map[common.Hash]uint64, len(state.Denied))
	for _, denied := range state.Denied {
		f.denied[denied.Hash] = denied.Number
	}
	f.deniedFacts = make(map[uint64]Fact, len(state.DeniedFacts))
	for _, denied := range state.DeniedFacts {
		f.deniedFacts[denied.Number] = denied.Fact
	}
	f.renderings = make(map[common.Hash]Rendering, len(state.Renderings))
	for _, rendering := range state.Renderings {
		f.renderings[rendering.Hash] = rendering.Rendering
	}
	f.replacementsByHash = make(map[common.Hash]Fact, len(state.Replacements))
	f.replacementsByNum = make(map[uint64]Fact, len(state.Replacements))
	for _, fact := range state.Replacements {
		f.replacementsByHash[fact.Hash] = fact
		f.replacementsByNum[fact.Number] = fact
	}
	f.rewindFacts = make(map[common.Hash]Fact, len(state.RewindFacts))
	for _, fact := range state.RewindFacts {
		f.rewindFacts[fact.Hash] = fact
	}
	return nil
}

// Flush atomically persists every externally observable part of the EL state. Ephemeral build jobs
// are intentionally excluded: after restart, stock op-node repeats forkchoiceUpdated to open them.
func (f *FactStore) Flush() error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.persist == nil {
		return nil
	}
	state := persistedState{
		Version:  persistenceVersion,
		Blocks:   f.blocks,
		Carriers: f.carriers,
		Cursors:  f.cursors,
		Tracker:  cloneTrackerState(f.tracker),
	}
	for hash, number := range f.denied {
		state.Denied = append(state.Denied, persistedDenial{Hash: hash, Number: number})
	}
	for number, fact := range f.deniedFacts {
		state.DeniedFacts = append(state.DeniedFacts, persistedDeniedFact{Number: number, Fact: fact})
	}
	for hash, rendering := range f.renderings {
		state.Renderings = append(state.Renderings, persistedRendering{Hash: hash, Rendering: rendering})
	}
	for _, fact := range f.replacementsByNum {
		state.Replacements = append(state.Replacements, fact)
	}
	for _, fact := range f.rewindFacts {
		state.RewindFacts = append(state.RewindFacts, fact)
	}
	raw, err := json.Marshal(&state)
	if err != nil {
		return fmt.Errorf("encode silhouette EL state: %w", err)
	}
	if err := f.persist.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(persistenceBucket).Put(persistenceKey, raw)
	}); err != nil {
		return fmt.Errorf("persist silhouette EL state: %w", err)
	}
	return nil
}

// Close writes one final checkpoint and releases the database lock.
func (f *FactStore) Close() error {
	if f.persist == nil {
		return nil
	}
	if err := f.Flush(); err != nil {
		return err
	}
	return f.persist.db.Close()
}

func cloneTrackerState(in trackerState) trackerState {
	out := in
	if in.Processed != nil {
		out.Processed = make(map[uint64]common.Hash, len(in.Processed))
		for number, hash := range in.Processed {
			out.Processed[number] = hash
		}
	}
	return out
}

func (f *FactStore) trackerState(start uint64) trackerState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if !f.tracker.Initialized || f.tracker.Start != start {
		return trackerState{Initialized: true, Start: start, Next: start, Processed: make(map[uint64]common.Hash)}
	}
	return cloneTrackerState(f.tracker)
}

func (f *FactStore) setTrackerState(state trackerState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tracker = cloneTrackerState(state)
}
