// Package el is the single construction seam for op-e2e's L2 execution layer.
// It selects between in-process op-geth and external-process op-reth based on
// the requested ELKind, so call sites never construct an EL backend directly.
// It lives apart from the geth/reth/services packages to avoid an import cycle.
package el

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/reth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
)

// L2Config carries everything needed to construct an L2 EL backend.
type L2Config struct {
	Kind    services.ELKind
	Name    string
	Genesis *core.Genesis
	// GenesisJSONPath is an op-reth-only alternative to Genesis.
	// For op-reth, set exactly one of these fields.
	GenesisJSONPath string
	JWTPath         string
	Logger          log.Logger
	// SequencerHTTP, when set, points non-sequencer nodes at the sequencer for
	// tx forwarding. On op-reth it maps to --rollup.sequencer-http.
	SequencerHTTP string
	// DataDir is the op-reth base directory. Callers must pass t.TempDir() so the
	// test framework owns cleanup. Ignored by the in-process op-geth backend.
	DataDir string
	// GethOptions are op-geth-only knobs. They cannot apply to op-reth; passing
	// any on the op-reth path is a hard error (see InitL2).
	GethOptions []geth.GethOption
}

// InitL2 constructs and starts the requested L2 EL backend, returning it as a
// services.EthInstance. The geth lifecycle (Node.Start) is folded in here;
// op-reth is already started and ready when reth.InitL2 returns.
func InitL2(ctx context.Context, cfg L2Config) (services.EthInstance, error) {
	if err := validateGenesisInput(cfg); err != nil {
		return nil, err
	}

	switch cfg.Kind {
	case services.ELKindOpGeth:
		gethOptions := append([]geth.GethOption{}, cfg.GethOptions...)
		if cfg.SequencerHTTP != "" {
			gethOptions = append(gethOptions, geth.WithSequencerHTTP(cfg.SequencerHTTP))
		}
		inst, err := geth.InitL2(cfg.Name, cfg.Genesis, cfg.JWTPath, gethOptions...)
		if err != nil {
			return nil, err
		}
		if err := inst.Node.Start(); err != nil {
			return nil, err
		}
		return inst, nil
	case services.ELKindOpReth:
		if len(cfg.GethOptions) > 0 {
			return nil, fmt.Errorf("L2 node %q requested op-reth but supplied %d geth options, which cannot apply to op-reth; pin op-geth for this test via e2esys.WithL2ELKind(services.ELKindOpGeth)", cfg.Name, len(cfg.GethOptions))
		}
		return reth.InitL2(ctx, cfg.Logger, cfg.Name, cfg.Genesis, cfg.JWTPath, reth.Config{
			GenesisJSONPath: cfg.GenesisJSONPath,
			SequencerHTTP:   cfg.SequencerHTTP,
			DataDir:         cfg.DataDir,
		})
	default:
		return nil, fmt.Errorf("unknown L2 EL kind %q for node %q", cfg.Kind, cfg.Name)
	}
}

func validateGenesisInput(cfg L2Config) error {
	switch cfg.Kind {
	case services.ELKindOpGeth:
		if cfg.GenesisJSONPath != "" {
			return fmt.Errorf("L2 node %q requested op-geth with a genesis JSON path; op-geth requires an in-memory genesis", cfg.Name)
		}
	case services.ELKindOpReth:
		hasGenesis := cfg.Genesis != nil
		hasGenesisPath := cfg.GenesisJSONPath != ""
		if hasGenesis == hasGenesisPath {
			inputCount := 0
			if hasGenesis {
				inputCount++
			}
			if hasGenesisPath {
				inputCount++
			}
			return fmt.Errorf("L2 node %q requested op-reth with %d genesis inputs; set exactly one of Genesis or GenesisJSONPath", cfg.Name, inputCount)
		}
	}
	return nil
}
