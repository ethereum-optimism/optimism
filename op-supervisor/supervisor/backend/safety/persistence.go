package safety

import (
	"encoding/gob"
	"os"
	"path"
)

// Persistence is an interface for loading and saving SafetyIndex objects.
type Persistence interface {
	Load() (SafetyIndex, error)
	Save(SafetyIndex) error
}

// simpleSafetyIndexPersistence is a simple implementation of the Persistence interface.
// it gob encodes/decodes the SafetyIndex to/from a file.
// more sophisticated implementations should maintain a database so that data from the SafetyIndex can be appended
type simpleSafetyIndexPersistence struct {
	filename string
	path     string
}

func NewSafetyIndexPersistence(path string) *simpleSafetyIndexPersistence {
	return &simpleSafetyIndexPersistence{
		filename: "safety_index.gob",
		path:     path,
	}
}

func (p *simpleSafetyIndexPersistence) Load() (SafetyIndex, error) {
	file, err := os.Open(path.Join(p.path, p.filename))
	if err != nil {
		return nil, err
	}
	var ret SafetyIndex
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (p *simpleSafetyIndexPersistence) Save(si *SafetyIndex) error {
	file, err := os.Create(path.Join(p.path, p.filename))
	if err != nil {
		return err
	}
	defer file.Close()
	// gob encode the SafetyIndex
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(si)
	if err != nil {
		return err
	}
	return nil
}
