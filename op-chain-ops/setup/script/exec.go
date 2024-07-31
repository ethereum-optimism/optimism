package script

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

// ComputeStateChecksum computes the hash of the JSON-serialized allocs, as checksum to sanity-check states with.
// Since states are cached by the script-hash that produced them, and users might modify them (they shouldn't),
// we cannot otherwise guarantee that the state is still accurate.
func ComputeStateChecksum(allocs *foundry.ForgeAllocs) common.Hash {
	data, err := json.Marshal(allocs)
	if err != nil {
		panic(fmt.Errorf("invalid allocs: %w", err))
	}
	return crypto.Keccak256Hash(data)
}

// CallResult bundles the results data of a call.
type CallResult struct {
	// Post state after executing this call.
	Post State

	// Labels added during this call (foundry vm.label cheatcode)
	LabelsDiff map[common.Address]string

	// Deployments are named contract deployments that were added during this call.
	DeploymentsDiff map[string]common.Address
}

// ExecCall executes a Call to produce a post State, and persists the necessary cache files.
func ExecCall(ctx context.Context, log log.Logger, cachePath string, pre State, scriptCall *Call) (result *CallResult, err error) {
	// Compute the cache key
	scriptHash := scriptCall.Hash()

	allocsCacheDir := filepath.Join(cachePath, "allocs")

	postStateCachePath := filepath.Join(allocsCacheDir, scriptHash.String()+".json")
	preStateCachePath := filepath.Join(allocsCacheDir, scriptCall.Prestate.String()+".json")
	diffCachePath := filepath.Join(cachePath, "diffs", scriptHash.String()+".json")

	// Check if we already have a diff.
	if _, err := os.Stat(diffCachePath); err != nil {
		if !os.IsNotExist(err) {
			log.Warn("Could not stat diff cache file, generating diff now", "err", err)
		}

		// Load the pre-state, we'll need it to compute the diff
		preAllocs, err := pre.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load pre-state allocs: %w", err)
		}

		// Check if pre-state exists, write if it does not.
		// We need it on disk for the script to operate on.
		if _, err := os.Stat(preStateCachePath); err != nil {
			if os.IsNotExist(err) {
				log.Info("Writing pre-state to disk", "file", preStateCachePath)
			} else {
				log.Warn("Could not stat pre-state cache file, writing pre-state now",
					"file", preStateCachePath, "err", err)
			}
			if err := foundry.WriteForgeAllocs(preStateCachePath, preAllocs); err != nil {
				return nil, fmt.Errorf("failed to write pre-state allocs: %w", err)
			}
		} else {
			log.Info("Pre-state already exists on disk, reusing it", "file", preStateCachePath)
		}

		// Run the script, to generate the post-state
		out, err := runForgeScript(ctx, log, preStateCachePath, postStateCachePath, scriptCall)
		if err != nil {
			return nil, fmt.Errorf("failed to execute forge script: %w", err)
		}

		// Load post-state
		postState := &CachedState{
			dir:        allocsCacheDir,
			scriptHash: scriptHash,
		}
		postAllocs, err := postState.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load post-state: %w", err)
		}
		// We compute a checksum of the diff,
		// so that later users of the diff can sanity-check the results match.
		postChecksum := ComputeStateChecksum(postAllocs)

		// Compute diff, to fill the cache with.
		allocsDiff, err := foundry.ComputeDiff(preAllocs, postAllocs)
		if err != nil {
			return nil, fmt.Errorf("failed to compute diff: %w", err)
		}

		diff := &CallDiff{
			Call:            scriptCall,
			Diff:            allocsDiff,
			LabelsDiff:      out.LabelsDiff,
			DeploymentsDiff: out.DeploymentsDiff,
			PostChecksum:    postChecksum,
		}
		if err := WriteCallDiff(diff, diffCachePath); err != nil {
			return nil, fmt.Errorf("failed to write diff: %w", err)
		}

		return &CallResult{
			Post:            postState,
			LabelsDiff:      diff.LabelsDiff,
			DeploymentsDiff: diff.DeploymentsDiff,
		}, nil
	} else {
		// Load the diff
		diff, err := LoadCallDiff(diffCachePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load diff: %w", err)
		}

		// Check if we already have the post-state.
		// If yes, then we don't have to load the pre-state, and don't have to apply the diff.
		if _, err := os.Stat(postStateCachePath); err != nil {
			if !os.IsNotExist(err) {
				log.Warn("Failed to check if post-state already exists. Reconstructing it now.", "err", err)
			}
		} else {
			log.Info("Detected existing post-state, re-using that.", "scriptHash", scriptHash)
			return &CallResult{
				Post: &CachedState{
					dir:        allocsCacheDir,
					scriptHash: scriptHash,
				},
				LabelsDiff:      diff.LabelsDiff,
				DeploymentsDiff: diff.DeploymentsDiff,
			}, nil
		}

		// Load the pre-state
		preAllocs, err := pre.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load pre-state allocs: %w", err)
		}

		// Compute post state by applying the diff
		postAllocs := foundry.ApplyDiff(preAllocs, diff.Diff)

		// Sanity check our post-state against the checksum in the diff
		postChecksum := ComputeStateChecksum(postAllocs)
		if postChecksum != diff.PostChecksum {
			return nil, fmt.Errorf("applying diff on prestate resulted in post-state with checksum %s, but expected checksum %s",
				postChecksum, diff.PostChecksum)
		}

		return &CallResult{
			Post: &InMemoryState{
				scriptHash: scriptHash,
				allocs:     postAllocs,
			},
			LabelsDiff:      diff.LabelsDiff,
			DeploymentsDiff: diff.DeploymentsDiff,
		}, nil
	}

}

type outWriter struct {
	log log.Logger
	lvl slog.Level
}

func (w *outWriter) Write(data []byte) (int, error) {
	w.log.Write(w.lvl, string(data))
	return len(data), nil
}

type forgeScriptOutput struct {

	// Labels added during this call (foundry vm.label cheatcode)
	LabelsDiff map[common.Address]string

	// Deployments are named contract deployments that were added during this call.
	DeploymentsDiff map[string]common.Address
}

func runForgeScript(ctx context.Context, logger log.Logger,
	preStateCachePath string, postStateCachePath string, call *Call) (out *forgeScriptOutput, err error) {
	// TODO: need to pass args:
	// preStateCachePath -> to load allocs from
	// postStateCachePath -> to write allocs to
	// call.Args -> patch DeployConfig contract values with this
	// call.Addrs -> patch Artifacts addresses with this
	// call.Prestate -> make it load this as initial state
	// call.Target -> magic identifier for deploy-script contract to call. Should be either the L1Deploy or L2Genesis script.
	// call.Sig -> call the same sig always, to handle the outer state loading/writing and patching of vars, but then call out to this method signature.
	scriptName := filepath.Join(
		"forge", "script", "../../../packages/contracts-bedrock/scripts/Incremental.s.sol",
		"--sender", "0x90F79bf6EB2c4f870365E785982E1f101E93b906") // i=3 in dev mnemonic, what devnet used to use to deploy.

	scriptCmd := exec.CommandContext(ctx, scriptName)
	// can use env
	scriptCmd.Env = []string{
		"TODO=hello",
	}
	// TODO: need to not write a "deployments" dir,
	//  and make sure the address/config patches actually avoid loading from a deployments dir.

	// collect the output in our logging
	scriptCmd.Stdout = &outWriter{lvl: log.LevelInfo, log: logger}
	scriptCmd.Stderr = &outWriter{lvl: log.LevelWarn, log: logger}

	if err := scriptCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}
	// TODO extract labels and deployments info

	return nil, nil
}
