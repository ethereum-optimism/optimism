package deployer

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

const faultGameAbsolutePrestateOverride = "faultGameAbsolutePrestate"

type PrestateConfig struct {
	Workdir  string
	Logger   log.Logger
	Prestate string
}

func (c *PrestateConfig) Check() error {
	if c.Workdir == "" {
		return fmt.Errorf("workdir must be specified")
	}
	if c.Logger == nil {
		return fmt.Errorf("logger must be specified")
	}
	if c.Prestate != "" {
		if _, err := parsePrestate(c.Prestate); err != nil {
			return fmt.Errorf("invalid %s: %w", PrestateFlagName, err)
		}
	}
	return nil
}

func PrestateCLI() func(cliCtx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(cliCtx)
		l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		oplog.SetGlobalLogHandler(l.Handler())

		ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)
		return Prestate(ctx, PrestateConfig{
			Workdir:  cliCtx.String(WorkdirFlagName),
			Logger:   l,
			Prestate: cliCtx.String(PrestateFlagName),
		})
	}
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

	flagPrestate, err := parseFlagPrestate(cfg.Prestate)
	if err != nil {
		return err
	}

	committed := 0
	for _, chain := range intent.Chains {
		intentPrestate, err := resolveIntentPrestate(chain, intent.GlobalDeployOverrides)
		if err != nil {
			return err
		}
		resolved, err := mergePrestateSources(chain.ID, flagPrestate, intentPrestate)
		if err != nil {
			return err
		}
		if !resolved.set {
			cfg.Logger.Info("skipping chain with no dispute absolute prestate source", "chainID", chain.ID.Hex())
			continue
		}

		if _, err := st.Chain(chain.ID); err != nil {
			return fmt.Errorf("chain %s has no prepared state entry; run op-deployer prepare before op-deployer prestate: %w", chain.ID.Hex(), err)
		}

		st.SetChainPrestate(chain.ID, resolved.hash)
		cfg.Logger.Info("committed dispute absolute prestate", "chainID", chain.ID.Hex(), "source", resolved.source, "prestate", resolved.hash.Hex())
		committed++
	}

	if committed == 0 {
		return fmt.Errorf("no prestates committed; provide --%s or set %s in the intent", PrestateFlagName, faultGameAbsolutePrestateOverride)
	}
	if err := pipeline.WriteState(cfg.Workdir, st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	return nil
}

type resolvedPrestate struct {
	set    bool
	source string
	raw    string
	hash   common.Hash
}

func parseFlagPrestate(raw string) (resolvedPrestate, error) {
	if raw == "" {
		return resolvedPrestate{}, nil
	}
	hash, err := parsePrestate(raw)
	if err != nil {
		return resolvedPrestate{}, fmt.Errorf("invalid %s: %w", PrestateFlagName, err)
	}
	return resolvedPrestate{
		set:    true,
		source: "flag",
		raw:    raw,
		hash:   hash,
	}, nil
}

func resolveIntentPrestate(chain *state.ChainIntent, globalOverrides map[string]any) (resolvedPrestate, error) {
	if raw, ok := chain.DeployOverrides[faultGameAbsolutePrestateOverride]; ok {
		return parseIntentPrestate(chain.ID, "chain-override", raw)
	}
	if raw, ok := globalOverrides[faultGameAbsolutePrestateOverride]; ok {
		return parseIntentPrestate(chain.ID, "global-override", raw)
	}
	return resolvedPrestate{}, nil
}

func parseIntentPrestate(chainID common.Hash, source string, raw any) (resolvedPrestate, error) {
	value, ok := raw.(string)
	if !ok {
		return resolvedPrestate{}, fmt.Errorf("%s %s for chain %s must be a string", source, faultGameAbsolutePrestateOverride, chainID.Hex())
	}
	hash, err := parsePrestate(value)
	if err != nil {
		return resolvedPrestate{}, fmt.Errorf("invalid %s %s for chain %s: %w", source, faultGameAbsolutePrestateOverride, chainID.Hex(), err)
	}
	return resolvedPrestate{
		set:    true,
		source: source,
		raw:    value,
		hash:   hash,
	}, nil
}

func mergePrestateSources(chainID common.Hash, flag, intent resolvedPrestate) (resolvedPrestate, error) {
	if flag.set && intent.set {
		if flag.hash != intent.hash {
			return resolvedPrestate{}, fmt.Errorf(
				"conflicting prestate sources for chain %s: flag=%s %s=%s",
				chainID.Hex(),
				flag.raw,
				intent.source,
				intent.raw,
			)
		}
		return intent, nil
	}
	if intent.set {
		return intent, nil
	}
	return flag, nil
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
