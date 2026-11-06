package filter

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

const failsafeStateFile = "failsafe.json"

// failsafeState is the manual failsafe state that the filter keeps across restarts.
type failsafeState struct {
	ManualFailsafe bool `json:"manualFailsafe"`
}

// FailsafeStatePath returns the path of the failsafe state file in dataDir.
// It returns an empty path when dataDir is empty, because the filter then uses temporary storage.
func FailsafeStatePath(dataDir string) string {
	if dataDir == "" {
		return ""
	}
	return filepath.Join(dataDir, failsafeStateFile)
}

// readManualFailsafe returns the stored manual failsafe state.
// A missing file or an empty path gives false.
func readManualFailsafe(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	state, err := jsonutil.LoadJSON[failsafeState](path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read failsafe state: %w", err)
	}
	return state.ManualFailsafe, nil
}

// initialManualFailsafe returns the manual failsafe state at start: on if the stored state or the start flag says so.
func initialManualFailsafe(path string, flagEnabled bool) (bool, error) {
	stored, err := readManualFailsafe(path)
	if err != nil {
		return false, err
	}
	return stored || flagEnabled, nil
}

func writeManualFailsafe(path string, enabled bool) error {
	if err := jsonutil.WriteJSON(failsafeState{ManualFailsafe: enabled}, ioutil.ToAtomicFile(path, 0o644)); err != nil {
		return fmt.Errorf("failed to write failsafe state: %w", err)
	}
	return nil
}
