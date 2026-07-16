package deployer

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

const (
	faultGameAbsolutePrestateOverride = "faultGameAbsolutePrestate"
	cannonFallbackPrestateOverride    = "cannonFallbackPrestate"
)

type PrestateConfig struct {
	Workdir                   string
	Logger                    log.Logger
	Prestate                  string
	PrestateSet               bool
	CannonFallbackPrestate    string
	CannonFallbackPrestateSet bool
}

func (c *PrestateConfig) Check() error {
	if c.Workdir == "" {
		return fmt.Errorf("workdir must be specified")
	}
	if c.Logger == nil {
		return fmt.Errorf("logger must be specified")
	}
	return nil
}

func newPrestateConfig(cliCtx *cli.Context, l log.Logger) PrestateConfig {
	return PrestateConfig{
		Workdir:                   cliCtx.String(WorkdirFlagName),
		Logger:                    l,
		Prestate:                  cliCtx.String(PrestateFlagName),
		PrestateSet:               cliCtx.IsSet(PrestateFlagName),
		CannonFallbackPrestate:    cliCtx.String(CannonFallbackPrestateFlagName),
		CannonFallbackPrestateSet: cliCtx.IsSet(CannonFallbackPrestateFlagName),
	}
}

func PrestateCLI() func(cliCtx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(cliCtx)
		l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		oplog.SetGlobalLogHandler(l.Handler())

		ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)
		return Prestate(ctx, newPrestateConfig(cliCtx, l))
	}
}

type prestateRole struct {
	name      string
	flagName  string
	intentKey string
}

var (
	selectedPrestateRole = prestateRole{
		name:      "selected prestate",
		flagName:  PrestateFlagName,
		intentKey: faultGameAbsolutePrestateOverride,
	}
	cannonFallbackPrestateRole = prestateRole{
		name:      "Cannon fallback prestate",
		flagName:  CannonFallbackPrestateFlagName,
		intentKey: cannonFallbackPrestateOverride,
	}
)

type prestateAssignment struct {
	ChainID        common.Hash
	GameType       embedded.GameType
	Selected       common.Hash
	CannonFallback common.Hash
}

func Prestate(ctx context.Context, cfg PrestateConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for prestate: %w", err)
	}

	intent, err := pipeline.ReadIntent(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}
	st, err := pipeline.ReadState(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	if !st.Prepared {
		return fmt.Errorf("state was not produced by op-deployer prepare; run op-deployer prepare before op-deployer prestate")
	}
	if err := pipeline.ValidateInputs(intent, st); err != nil {
		return fmt.Errorf("failed to validate prestate inputs: %w", err)
	}

	selectedCommand, err := parseCommandPrestate(selectedPrestateRole, cfg.PrestateSet || cfg.Prestate != "", cfg.Prestate)
	if err != nil {
		return err
	}
	fallbackCommand, err := parseCommandPrestate(cannonFallbackPrestateRole, cfg.CannonFallbackPrestateSet || cfg.CannonFallbackPrestate != "", cfg.CannonFallbackPrestate)
	if err != nil {
		return err
	}

	assignments := make([]prestateAssignment, 0, len(intent.Chains))
	hasCannonKona := false
	hasSuperCannonKona := false
	for _, chain := range intent.Chains {
		gameType, err := resolveInitialGameType(intent, chain)
		if err != nil {
			return err
		}

		assignment := prestateAssignment{ChainID: chain.ID, GameType: gameType}
		switch gameType {
		case embedded.GameTypePermissionedCannon:
			// Clear stale prestate commitments.
		case embedded.GameTypeCannonKona:
			hasCannonKona = true
			selected, err := resolvePrestateRole(chain, intent.GlobalDeployOverrides, selectedPrestateRole, selectedCommand)
			if err != nil {
				return err
			}
			fallback, err := resolvePrestateRole(chain, intent.GlobalDeployOverrides, cannonFallbackPrestateRole, fallbackCommand)
			if err != nil {
				return err
			}
			if !selected.set {
				return fmt.Errorf("chain %s with CANNON_KONA requires %s from --%s or intent key %s", chain.ID.Hex(), selectedPrestateRole.name, selectedPrestateRole.flagName, selectedPrestateRole.intentKey)
			}
			if !fallback.set {
				return fmt.Errorf("chain %s with CANNON_KONA requires %s from --%s or intent key %s", chain.ID.Hex(), cannonFallbackPrestateRole.name, cannonFallbackPrestateRole.flagName, cannonFallbackPrestateRole.intentKey)
			}
			if selected.hash == fallback.hash {
				return fmt.Errorf("chain %s with CANNON_KONA requires different selected and Cannon fallback prestates; both resolve to %s", chain.ID.Hex(), selected.hash.Hex())
			}
			assignment.Selected = selected.hash
			assignment.CannonFallback = fallback.hash
		case embedded.GameTypeSuperCannonKona:
			hasSuperCannonKona = true
			selected, err := resolvePrestateRole(chain, intent.GlobalDeployOverrides, selectedPrestateRole, selectedCommand)
			if err != nil {
				return err
			}
			if !selected.set {
				return fmt.Errorf("chain %s with SUPER_CANNON_KONA requires %s from --%s or intent key %s", chain.ID.Hex(), selectedPrestateRole.name, selectedPrestateRole.flagName, selectedPrestateRole.intentKey)
			}
			assignment.Selected = selected.hash
		default:
			return fmt.Errorf("chain %s has unsupported initial dispute game type %d", chain.ID.Hex(), gameType)
		}
		assignments = append(assignments, assignment)
	}

	if hasCannonKona && hasSuperCannonKona {
		return fmt.Errorf("an intent cannot mix CANNON_KONA and SUPER_CANNON_KONA initial games")
	}
	if hasSuperCannonKona && len(assignments) > 1 {
		return fmt.Errorf("SUPER_CANNON_KONA is not supported in a multi-chain intent")
	}
	// Reject prestate flags that would otherwise be silently ignored.
	if selectedCommand.set && !hasCannonKona && !hasSuperCannonKona {
		return fmt.Errorf("--%s was supplied but no chain resolves to a game type that uses the %s; check respectedGameType in the intent", selectedPrestateRole.flagName, selectedPrestateRole.name)
	}
	if fallbackCommand.set && !hasCannonKona {
		return fmt.Errorf("--%s was supplied but no chain resolves to CANNON_KONA; check respectedGameType in the intent", cannonFallbackPrestateRole.flagName)
	}

	// Avoid partial updates if a chain is missing.
	for _, assignment := range assignments {
		if _, err := st.Chain(assignment.ChainID); err != nil {
			return fmt.Errorf("run op-deployer prepare before op-deployer prestate for chain %s: %w", assignment.ChainID.Hex(), err)
		}
	}

	for _, assignment := range assignments {
		chainState, err := st.Chain(assignment.ChainID)
		if err != nil {
			return fmt.Errorf("prepared chain %s disappeared: %w", assignment.ChainID.Hex(), err)
		}
		chainState.Prestate = assignment.Selected
		chainState.CannonFallbackPrestate = assignment.CannonFallback
	}

	if err := pipeline.WriteState(cfg.Workdir, st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	for _, assignment := range assignments {
		switch assignment.GameType {
		case embedded.GameTypeCannonKona:
			cfg.Logger.Info("committed deployment prestates", "chainID", assignment.ChainID.Hex(), "selected", assignment.Selected.Hex(), "cannonFallback", assignment.CannonFallback.Hex())
		case embedded.GameTypeSuperCannonKona:
			cfg.Logger.Info("committed deployment prestate", "chainID", assignment.ChainID.Hex(), "selected", assignment.Selected.Hex())
		}
	}
	return nil
}

type resolvedPrestate struct {
	set    bool
	source string
	raw    string
	hash   common.Hash
}

func parseCommandPrestate(role prestateRole, set bool, raw string) (resolvedPrestate, error) {
	if !set {
		return resolvedPrestate{}, nil
	}
	hash, err := parsePrestate(raw)
	if err != nil {
		return resolvedPrestate{}, fmt.Errorf("invalid --%s %s: %w", role.flagName, role.name, err)
	}
	return resolvedPrestate{set: true, source: "command/environment", raw: raw, hash: hash}, nil
}

func resolvePrestateRole(chain *state.ChainIntent, globalOverrides map[string]any, role prestateRole, command resolvedPrestate) (resolvedPrestate, error) {
	intentValue, err := resolveIntentPrestate(chain, globalOverrides, role)
	if err != nil {
		return resolvedPrestate{}, err
	}
	if command.set && intentValue.set {
		if command.hash != intentValue.hash {
			return resolvedPrestate{}, fmt.Errorf(
				"conflicting %s sources for chain %s: --%s=%s %s %s=%s",
				role.name,
				chain.ID.Hex(),
				role.flagName,
				command.raw,
				intentValue.source,
				role.intentKey,
				intentValue.raw,
			)
		}
		return intentValue, nil
	}
	if intentValue.set {
		return intentValue, nil
	}
	return command, nil
}

func resolveIntentPrestate(chain *state.ChainIntent, globalOverrides map[string]any, role prestateRole) (resolvedPrestate, error) {
	if raw, ok := chain.DeployOverrides[role.intentKey]; ok {
		return parseIntentPrestate(chain.ID, "chain override", raw, role)
	}
	if raw, ok := globalOverrides[role.intentKey]; ok {
		return parseIntentPrestate(chain.ID, "global override", raw, role)
	}
	return resolvedPrestate{}, nil
}

func parseIntentPrestate(chainID common.Hash, source string, raw any, role prestateRole) (resolvedPrestate, error) {
	value, ok := raw.(string)
	if !ok {
		return resolvedPrestate{}, fmt.Errorf("%s %s for chain %s (%s) must be a string", source, role.intentKey, chainID.Hex(), role.name)
	}
	hash, err := parsePrestate(value)
	if err != nil {
		return resolvedPrestate{}, fmt.Errorf("invalid %s %s for chain %s (%s): %w", source, role.intentKey, chainID.Hex(), role.name, err)
	}
	return resolvedPrestate{set: true, source: source, raw: value, hash: hash}, nil
}

func resolveInitialGameType(intent *state.Intent, chain *state.ChainIntent) (embedded.GameType, error) {
	type initialGameConfig struct {
		GameType uint32 `json:"respectedGameType"`
	}
	cfg, err := jsonutil.MergeJSON(
		initialGameConfig{GameType: standard.DisputeGameType},
		intent.GlobalDeployOverrides,
		chain.DeployOverrides,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve initial dispute game type for chain %s: %w", chain.ID.Hex(), err)
	}
	return embedded.GameType(cfg.GameType), nil
}

func parsePrestate(raw string) (common.Hash, error) {
	if !strings.HasPrefix(raw, "0x") {
		return common.Hash{}, fmt.Errorf("must start with 0x")
	}

	hexValue := raw[2:]
	if len(hexValue) != common.HashLength*2 {
		return common.Hash{}, fmt.Errorf("must contain exactly 64 hex characters after 0x")
	}

	decoded, err := hex.DecodeString(hexValue)
	if err != nil {
		return common.Hash{}, fmt.Errorf("must be valid hex: %w", err)
	}

	hash := common.BytesToHash(decoded)
	if hash == (common.Hash{}) {
		return common.Hash{}, fmt.Errorf("must not be zero")
	}
	return hash, nil
}
