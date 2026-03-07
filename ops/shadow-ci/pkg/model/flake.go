package model

import (
	"encoding/json"
	"os"
	"time"
)

// StateStore is a minimal interface for loading/saving state blobs.
// This avoids a circular import with the state package.
type StateStore interface {
	Load(key string) ([]byte, error)
	Save(key string, data []byte) error
}

// FlakeState represents a test's position in the flake lifecycle.
type FlakeState string

const (
	FlakeHealthy     FlakeState = "healthy"
	FlakeSuspected   FlakeState = "suspected"
	FlakeQuarantined FlakeState = "quarantined"
	FlakeShaking     FlakeState = "shaking"
	FlakeDiagnosed   FlakeState = "diagnosed"
	FlakeFixed       FlakeState = "fixed"
	FlakeAccepted    FlakeState = "accepted"
)

// FlakeSeverity determines how aggressively a flake is handled.
type FlakeSeverity string

const (
	SeverityLow      FlakeSeverity = "low"
	SeverityMedium   FlakeSeverity = "medium"
	SeverityHigh     FlakeSeverity = "high"
	SeverityCritical FlakeSeverity = "critical"
)

// FlakeRecord tracks a test through the flake lifecycle.
type FlakeRecord struct {
	TestKey     string        `json:"test_key"`
	Language    string        `json:"language"`
	Fingerprint string       `json:"fingerprint"`

	State    FlakeState    `json:"state"`
	Severity FlakeSeverity `json:"severity"`

	FlakeCount     int       `json:"flake_count"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
	StateChangedAt time.Time `json:"state_changed_at"`

	DiagnosisNote string `json:"diagnosis_note,omitempty"`
	FixPR         int    `json:"fix_pr,omitempty"`
}

// FlakeDB is the in-memory flake state database.
type FlakeDB struct {
	Records map[string]*FlakeRecord `json:"records"`
}

// NewFlakeDB creates an empty FlakeDB.
func NewFlakeDB() *FlakeDB {
	return &FlakeDB{Records: make(map[string]*FlakeRecord)}
}

// IsQuarantined returns true if the given test key is currently quarantined.
func (db *FlakeDB) IsQuarantined(testKey string) bool {
	if db == nil {
		return false
	}
	r, ok := db.Records[testKey]
	return ok && r.State == FlakeQuarantined
}

// QuarantinedKeys returns all test keys currently in quarantine.
func (db *FlakeDB) QuarantinedKeys() []string {
	if db == nil {
		return nil
	}
	var keys []string
	for k, r := range db.Records {
		if r.State == FlakeQuarantined {
			keys = append(keys, k)
		}
	}
	return keys
}

// LoadFlakeDB reads a FlakeDB from a JSON file. Returns an empty DB if the file doesn't exist.
func LoadFlakeDB(path string) (*FlakeDB, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewFlakeDB(), nil
		}
		return nil, err
	}
	db := NewFlakeDB()
	if err := json.Unmarshal(data, db); err != nil {
		return nil, err
	}
	return db, nil
}

// Save writes the FlakeDB to a JSON file.
func (db *FlakeDB) Save(path string) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadFlakeDBFromStore loads a FlakeDB from a StateStore. Returns an empty DB
// if the key doesn't exist.
func LoadFlakeDBFromStore(store StateStore, key string) (*FlakeDB, error) {
	data, err := store.Load(key)
	if err != nil {
		// Treat not-found as empty DB.
		return NewFlakeDB(), nil
	}
	db := NewFlakeDB()
	if err := json.Unmarshal(data, db); err != nil {
		return nil, err
	}
	return db, nil
}

// SaveToStore writes the FlakeDB to a StateStore.
func (db *FlakeDB) SaveToStore(store StateStore, key string) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return store.Save(key, data)
}
