package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	bolt "go.etcd.io/bbolt"
)

// recoveryJournal retains hash-linked private headers used to authenticate a
// surviving prefix. Rewind can remove these headers from the EL, so commit must
// complete before changing forkchoice. Entries below private finality are pruned.
// Only private headers are stored; transaction bodies remain in the private EL.
type recoveryJournal struct {
	path    string
	genesis common.Hash
	headers map[common.Hash]eth.L2BlockRef
}

func (j *recoveryJournal) load() error {
	if j.headers != nil {
		return nil
	}
	headers := make(map[common.Hash]eth.L2BlockRef)
	if j.path == "" {
		j.headers = headers
		return nil
	}
	if _, err := os.Stat(j.path); os.IsNotExist(err) {
		j.headers = headers
		return nil
	} else if err != nil {
		return err
	}
	db, err := bolt.Open(j.path, 0600, &bolt.Options{ReadOnly: true, Timeout: time.Second})
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(j.genesis[:])
		if b == nil {
			return fmt.Errorf("recovery journal belongs to another private genesis")
		}
		return b.ForEach(func(k, v []byte) error {
			var ref eth.L2BlockRef
			if err := json.Unmarshal(v, &ref); err != nil {
				return err
			}
			if len(k) != common.HashLength || common.BytesToHash(k) != ref.Hash {
				return fmt.Errorf("inconsistent recovery journal header")
			}
			headers[ref.Hash] = ref
			return nil
		})
	})
	if err != nil {
		return err
	}
	j.headers = headers
	return nil
}

func (j *recoveryJournal) commit(finalized uint64) error {
	if j == nil || j.path == "" {
		return fmt.Errorf("private prefix recovery requires --l2.follow.source.recovery-path")
	}
	if err := j.load(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(j.path), 0700); err != nil {
		return err
	}
	db, err := bolt.Open(j.path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return err
	}
	defer db.Close()
	// bbolt fsyncs the atomic transaction before returning. A crash cannot leave
	// forkchoice rewound with only a partially written ancestry record.
	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(j.genesis[:])
		if err != nil {
			return err
		}
		for hash, ref := range j.headers {
			if ref.Number < finalized {
				if err := b.Delete(hash[:]); err != nil {
					return err
				}
				delete(j.headers, hash)
				continue
			}
			data, err := json.Marshal(ref)
			if err != nil {
				return err
			}
			if err := b.Put(hash[:], data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Persist the directory entry too when this was the first recovery.
	dir, err := os.Open(filepath.Dir(j.path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()

}

func (f *followRecovery) privateHeader(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error) {
	if f.journal != nil {
		if err := f.journal.load(); err != nil {
			return eth.L2BlockRef{}, err
		}
		if ref, ok := f.journal.headers[hash]; ok {
			return ref, nil
		}
	}
	ref, err := f.l2.L2BlockRefByHash(ctx, hash)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if ref.Hash != hash {
		return eth.L2BlockRef{}, fmt.Errorf("private header hash does not match request")
	}
	if f.journal != nil {
		f.journal.headers[hash] = ref
	}
	return ref, nil
}
