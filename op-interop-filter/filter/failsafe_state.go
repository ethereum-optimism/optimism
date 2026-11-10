package filter

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum-optimism/optimism/op-service/log"
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
// An unreadable state file stops the start unless the flag is set. An active failsafe blocks all interop
// transactions, so the filter must not enable it on a guess.
func initialManualFailsafe(logger log.Logger, path string, flagEnabled bool) (bool, error) {
	if flagEnabled {
		if _, err := readManualFailsafe(path); err != nil {
			logger.Error("Ignoring unreadable failsafe state because the failsafe flag is set", "path", path, "err", err)
		}
		return true, nil
	}
	enabled, err := readManualFailsafe(path)
	if err != nil {
		return false, fmt.Errorf("%w (set --failsafe-enabled to ignore the file and start with the failsafe on)", err)
	}
	return enabled, nil
}

func writeManualFailsafe(path string, enabled bool) error {
	if err := jsonutil.WriteJSON(failsafeState{ManualFailsafe: enabled}, ioutil.ToAtomicFile(path, 0o644)); err != nil {
		return fmt.Errorf("failed to write failsafe state: %w", err)
	}
	return nil
}
