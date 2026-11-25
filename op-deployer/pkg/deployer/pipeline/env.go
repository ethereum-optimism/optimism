package pipeline

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	opgenesis "github.com/ethereum-optimism/optimism/op-core/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/log"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

type Env struct {
	StateWriter StateWriter
	L1Client    *ethclient.Client
	Deployer    common.Address
	Logger      log.Logger
	// Forge runs the deploy scripts.
	Forge *opcm.ForgeEnv
	// L2Forge runs the L2 genesis script in the L2 contracts.
	L2Forge *forge.Client
	// IsGenesis identifies deployments that build genesis allocations instead of targeting a running chain.
	IsGenesis bool
	// DeployMockSP1Verifier explicitly opts development environments into an accept-all raw verifier.
	DeployMockSP1Verifier bool
	// AllowUnoptimizedContracts lets the L2 genesis hold contracts over the EIP-170 code-size limit.
	// Genesis deployments set it, as does the test-only ApplyPipelineOpts.AllowUnoptimizedContracts.
	AllowUnoptimizedContracts bool
	Context                   context.Context
}

type StateWriter interface {
	WriteState(st *state.State) error
}

type stateWriterFunc func(st *state.State) error

func (f stateWriterFunc) WriteState(st *state.State) error {
	return f(st)
}

func WorkdirStateWriter(workdir string) StateWriter {
	return stateWriterFunc(func(st *state.State) error {
		return WriteState(workdir, st)
	})
}

func NoopStateWriter() StateWriter {
	return stateWriterFunc(func(st *state.State) error {
		return nil
	})
}

// ReadIntent reads the workdir's intent.toml. Unknown keys are ignored except under
// l1DevGenesisParams.blobSchedule, where dropping a fork's entry would silently reprice its blobs.
func ReadIntent(workdir string) (*state.Intent, error) {
	intent, undecoded, err := jsonutil.LoadTOMLWithUndecoded[state.Intent](path.Join(workdir, "intent.toml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read intent file: %w", err)
	}
	if err := checkBlobScheduleKeys(undecoded); err != nil {
		return nil, fmt.Errorf("failed to read intent file: %w", err)
	}
	return intent, nil
}

// checkBlobScheduleKeys returns an error that names each unknown key under
// l1DevGenesisParams.blobSchedule, if there are any.
func checkBlobScheduleKeys(undecoded []toml.Key) error {
	schedule := reflect.TypeFor[params.BlobScheduleConfig]()
	var unknown []string
	for _, key := range undecoded {
		// Compare per key part with EqualFold, as the decoder matches key parts to fields.
		if len(key) <= 2 || !strings.EqualFold(key[0], "l1DevGenesisParams") || !strings.EqualFold(key[1], "blobSchedule") {
			continue
		}
		// Every key below an unknown table is undecoded too, and the table itself is not listed
		// when it is only implied by a dotted key. Report the first key part that names no field.
		t := schedule
		for i := 2; i < len(key) && t.Kind() == reflect.Struct; i++ {
			field, ok := t.FieldByNameFunc(func(name string) bool { return strings.EqualFold(name, key[i]) })
			if !ok {
				key = key[:i+1]
				break
			}
			t = field.Type
			if t.Kind() == reflect.Pointer {
				t = t.Elem()
			}
		}
		name := key.String()
		// The two unknown keys an intent is likely to hold: a fork entry and a blob config field.
		switch last := key[len(key)-1]; {
		case len(key) == 3 && strings.EqualFold(last, "osaka"):
			name += " (Osaka has no entry of its own and uses prague)"
		case len(key) == 4 && strings.EqualFold(last, "baseFeeUpdateFraction"):
			name += " (the TOML key is updateFraction)"
		}
		if !slices.Contains(unknown, name) {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	entries := make([]string, schedule.NumField())
	for i := range entries {
		entries[i] = strings.ToLower(schedule.Field(i).Name)
	}
	return fmt.Errorf("unknown blob schedule keys: %s; valid entries are %s, each with target, max and updateFraction",
		strings.Join(unknown, ", "), strings.Join(entries, ", "))
}

func ReadState(workdir string) (*state.State, error) {
	statePath := path.Join(workdir, "state.json")
	st, err := jsonutil.LoadJSON[state.State](statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	return st, nil
}

func WriteState(workdir string, st *state.State) error {
	statePath := path.Join(workdir, "state.json")
	return st.WriteToFile(statePath)
}

type Stage func(ctx context.Context, env *Env, bundle artifacts.Bundle, intent *state.Intent, st *state.State) error

func RenderGenesisAndRollup(globalState *state.State, chainID common.Hash, useGlobalIntent *state.Intent) (*opgenesis.Genesis, *rollup.Config, error) {
	if useGlobalIntent == nil && globalState.AppliedIntent == nil {
		return nil, nil, fmt.Errorf("chain state is not applied - run op-deployer apply")
	}

	globalIntent := useGlobalIntent
	if globalIntent == nil {
		globalIntent = globalState.AppliedIntent
	}

	chainIntent, err := globalIntent.Chain(chainID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get applied chain intent: %w", err)
	}

	chainState, err := globalState.Chain(chainID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chain ID %s: %w", chainID.String(), err)
	}

	if chainState.StartBlock == nil {
		return nil, nil, fmt.Errorf("chain %s has no pinned start block - run op-deployer prepare first", chainID.Hex())
	}

	l2Allocs := chainState.Allocs.Data
	config, err := state.CombineDeployConfig(
		globalIntent,
		chainIntent,
		globalState,
		chainState,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to combine L2 init config: %w", err)
	}

	l2GenesisBuilt, err := genesis.BuildL2Genesis(&config, l2Allocs, chainState.StartBlock.ToBlockRef())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build L2 genesis: %w", err)
	}
	l2GenesisBlock := l2GenesisBuilt.ToBlock()

	// Cross-check against the hash prepare committed.
	if chainState.GenesisBlockHash != nil && l2GenesisBlock.Hash() != *chainState.GenesisBlockHash {
		return nil, nil, fmt.Errorf(
			"chain %s: rendered genesis block hash %s does not match the committed GenesisBlockHash %s from prepare; rerun op-deployer prepare",
			chainID.Hex(), l2GenesisBlock.Hash(), *chainState.GenesisBlockHash,
		)
	}

	rollupConfig, err := config.RollupConfig(
		chainState.StartBlock.ToBlockRef(),
		l2GenesisBlock.Hash(),
		bigs.Uint64Strict(l2GenesisBlock.Number()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build rollup config: %w", err)
	}

	if err := rollupConfig.Check(); err != nil {
		return nil, nil, fmt.Errorf("generated rollup config does not pass validation: %w", err)
	}

	return l2GenesisBuilt, rollupConfig, nil
}

// ResolveRenderIntent returns the intent to use when rendering derived artifacts for a chain.
// Applied state always uses the frozen AppliedIntent snapshot, so rendered
// artifacts match what was actually broadcast on-chain. Prepared uses the snapshot
// stored in PreparedDeployment.
func ResolveRenderIntent(workdir string, globalState *state.State) (*state.Intent, error) {
	if globalState.AppliedIntent != nil {
		return globalState.AppliedIntent, nil
	}
	if globalState.PreparedDeployment == nil {
		return nil, fmt.Errorf("chain state is neither prepared nor applied - run op-deployer prepare or apply")
	}

	return globalState.PreparedDeployment.Intent, nil
}
