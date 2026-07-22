package artifacts

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"

	"github.com/ethereum/go-ethereum/common"
)

type integrityChecker interface {
	CheckIntegrity(data []byte) error
}

type hashIntegrityChecker struct {
	hash common.Hash
}

func (h *hashIntegrityChecker) CheckIntegrity(data []byte) error {
	hash := sha256.Sum256(data)
	if hash != h.hash {
		return fmt.Errorf("integrity check failed - expected: %x, got: %x", h.hash, hash)
	}
	return nil
}

type noopIntegrityChecker struct{}

func (noopIntegrityChecker) CheckIntegrity([]byte) error {
	return nil
}

// ContentDigest hashes the file paths and contents in a bundle.
// It ignores empty directories and file metadata so extraction does not change the hash.
func ContentDigest(bundle fs.FS) (common.Hash, error) {
	digest := sha256.New()
	err := fs.WalkDir(bundle, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		file, err := bundle.Open(name)
		if err != nil {
			return err
		}

		content := sha256.New()
		_, copyErr := io.Copy(content, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if err := binary.Write(digest, binary.BigEndian, uint64(len(name))); err != nil {
			return err
		}
		if _, err := io.WriteString(digest, name); err != nil {
			return err
		}
		_, err = digest.Write(content.Sum(nil))
		return err
	})
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to digest artifact bundle: %w", err)
	}
	return common.BytesToHash(digest.Sum(nil)), nil
}
