package batcher

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
)

// PrivateInteropCLIConfig is the --private-interop.* flag group as read from the CLI.
//
// It holds the flags' RAW values — strings for addresses and hashes — and does every conversion in
// Check, so that a malformed address is a startup error naming the flag rather than a zero value
// that only fails once a range is being built. Resolve returns the typed form, and never runs
// unless Check has passed.
//
// The group is inert unless Enabled. Enabled, it is ALL-OR-NOTHING: there is no half-configured
// Private Interop batcher, because every field below is load-bearing for bytes that go on L1.
type PrivateInteropCLIConfig struct {
	Enabled bool

	// RenderingRollupConfigPath is the path to the RENDERING's rollup.json.
	RenderingRollupConfigPath string
	// RenderingRPC is the execution client following the rendering — the parent-check follower.
	RenderingRPC string

	// MaxBlocksPerRange is the cadence.
	MaxBlocksPerRange uint64

	// ClaimRegistry, EventReplayer and ReplayMessenger are the rendering's genesis-assigned
	// addresses, as hex strings.
	ClaimRegistry   string
	EventReplayer   string
	ReplayMessenger string
	// ExtraEmitters are additional emitter addresses, as hex strings.
	ExtraEmitters []string

	// RollupConfigHash and DepSetHash are the claim's two configuration commitments, as hex.
	RollupConfigHash string
	DepSetHash       string

	// OperatorKey is the operator EOA's private key, as hex.
	OperatorKey string

	GasLimitExport uint64
	GasLimitImport uint64
	GasLimitEvent  uint64
	GasLimitClaim  uint64
	GasFeeCap      uint64
	GasTipCap      uint64
}

// ReadPrivateInteropCLIConfig parses the flag group.
func ReadPrivateInteropCLIConfig(ctx *cli.Context) PrivateInteropCLIConfig {
	return PrivateInteropCLIConfig{
		Enabled:                   ctx.Bool(flags.PrivateInteropEnabledFlag.Name),
		RenderingRollupConfigPath: ctx.String(flags.PrivateInteropRenderingRollupConfigFlag.Name),
		RenderingRPC:              ctx.String(flags.PrivateInteropRenderingRPCFlag.Name),
		MaxBlocksPerRange:         ctx.Uint64(flags.PrivateInteropMaxBlocksPerRangeFlag.Name),
		ClaimRegistry:             ctx.String(flags.PrivateInteropClaimRegistryFlag.Name),
		EventReplayer:             ctx.String(flags.PrivateInteropEventReplayerFlag.Name),
		ReplayMessenger:           ctx.String(flags.PrivateInteropReplayMessengerFlag.Name),
		ExtraEmitters:             ctx.StringSlice(flags.PrivateInteropExtraEmittersFlag.Name),
		RollupConfigHash:          ctx.String(flags.PrivateInteropRollupConfigHashFlag.Name),
		DepSetHash:                ctx.String(flags.PrivateInteropDepSetHashFlag.Name),
		OperatorKey:               ctx.String(flags.PrivateInteropOperatorKeyFlag.Name),
		GasLimitExport:            ctx.Uint64(flags.PrivateInteropGasLimitExportFlag.Name),
		GasLimitImport:            ctx.Uint64(flags.PrivateInteropGasLimitImportFlag.Name),
		GasLimitEvent:             ctx.Uint64(flags.PrivateInteropGasLimitEventFlag.Name),
		GasLimitClaim:             ctx.Uint64(flags.PrivateInteropGasLimitClaimFlag.Name),
		GasFeeCap:                 ctx.Uint64(flags.PrivateInteropGasFeeCapFlag.Name),
		GasTipCap:                 ctx.Uint64(flags.PrivateInteropGasTipCapFlag.Name),
	}
}

// minRenderingTxGas is the intrinsic gas of a transaction with calldata. A gas limit under it
// cannot pay for the transaction's own existence, so it is a configuration error rather than an
// under-provisioned replay.
const minRenderingTxGas = 21_000

// Check validates the whole group, and is the only place that decides what a valid Private Interop
// configuration is.
//
// Every address and hash is checked for the ZERO value as well as for syntax, matching the posture
// render/abi.go takes: the rendering's contracts are placed by its genesis builder, so their
// addresses are per-deployment configuration with no defensible default, and sending a claim to the
// zero address is a rendering that silently lost its record. Loud at startup beats a reverted
// transaction on a live chain.
func (c *PrivateInteropCLIConfig) Check() error {
	if !c.Enabled {
		return nil
	}
	if c.RenderingRollupConfigPath == "" {
		return errors.New("private interop: --private-interop.rendering-rollup-config is required")
	}
	if c.RenderingRPC == "" {
		return errors.New("private interop: --private-interop.rendering-rpc is required")
	}
	if c.MaxBlocksPerRange == 0 {
		return errors.New("private interop: --private-interop.max-blocks-per-range must be greater than zero")
	}
	for _, f := range []struct {
		flag, value string
	}{
		{flags.PrivateInteropClaimRegistryFlag.Name, c.ClaimRegistry},
		{flags.PrivateInteropEventReplayerFlag.Name, c.EventReplayer},
		{flags.PrivateInteropReplayMessengerFlag.Name, c.ReplayMessenger},
	} {
		if _, err := parseAddress(f.flag, f.value); err != nil {
			return err
		}
	}
	for _, a := range c.ExtraEmitters {
		if _, err := parseAddress(flags.PrivateInteropExtraEmittersFlag.Name, a); err != nil {
			return err
		}
	}
	for _, f := range []struct {
		flag, value string
	}{
		{flags.PrivateInteropRollupConfigHashFlag.Name, c.RollupConfigHash},
		{flags.PrivateInteropDepSetHashFlag.Name, c.DepSetHash},
	} {
		if _, err := parseHash(f.flag, f.value); err != nil {
			return err
		}
	}
	if _, err := c.operatorKey(); err != nil {
		return err
	}
	for _, f := range []struct {
		flag  string
		limit uint64
	}{
		{flags.PrivateInteropGasLimitExportFlag.Name, c.GasLimitExport},
		{flags.PrivateInteropGasLimitImportFlag.Name, c.GasLimitImport},
		{flags.PrivateInteropGasLimitEventFlag.Name, c.GasLimitEvent},
		{flags.PrivateInteropGasLimitClaimFlag.Name, c.GasLimitClaim},
	} {
		if f.limit < minRenderingTxGas {
			return fmt.Errorf("private interop: --%s is %d, below the %d intrinsic gas a transaction costs",
				f.flag, f.limit, minRenderingTxGas)
		}
	}
	if c.GasFeeCap == 0 {
		return errors.New("private interop: --private-interop.gas-fee-cap must be greater than zero; a zero-priced rendering transaction is never included")
	}
	if c.GasTipCap > c.GasFeeCap {
		return fmt.Errorf("private interop: --private-interop.gas-tip-cap (%d) exceeds --private-interop.gas-fee-cap (%d)", c.GasTipCap, c.GasFeeCap)
	}
	return nil
}

// PrivateInteropSettings is the group in its typed form.
type PrivateInteropSettings struct {
	RenderingRollupConfigPath string
	RenderingRPC              string
	MaxBlocksPerRange         uint64

	ClaimRegistry   common.Address
	EventReplayer   common.Address
	ReplayMessenger common.Address
	Emitters        render.EmitterSet

	RollupConfigHash common.Hash
	DepSetHash       common.Hash

	OperatorKey *ecdsa.PrivateKey
	Gas         render.GasPolicy
}

// Resolve converts the raw group into its typed form. It re-runs Check first, so a caller cannot
// resolve a configuration nobody validated.
func (c *PrivateInteropCLIConfig) Resolve() (*PrivateInteropSettings, error) {
	if err := c.Check(); err != nil {
		return nil, err
	}
	if !c.Enabled {
		return nil, errors.New("private interop: not enabled")
	}
	registry, _ := parseAddress(flags.PrivateInteropClaimRegistryFlag.Name, c.ClaimRegistry)
	replayer, _ := parseAddress(flags.PrivateInteropEventReplayerFlag.Name, c.EventReplayer)
	messenger, _ := parseAddress(flags.PrivateInteropReplayMessengerFlag.Name, c.ReplayMessenger)
	extra := make([]common.Address, 0, len(c.ExtraEmitters))
	for _, a := range c.ExtraEmitters {
		addr, _ := parseAddress(flags.PrivateInteropExtraEmittersFlag.Name, a)
		extra = append(extra, addr)
	}
	rollupConfigHash, _ := parseHash(flags.PrivateInteropRollupConfigHashFlag.Name, c.RollupConfigHash)
	depSetHash, _ := parseHash(flags.PrivateInteropDepSetHashFlag.Name, c.DepSetHash)
	key, err := c.operatorKey()
	if err != nil {
		return nil, err
	}
	return &PrivateInteropSettings{
		RenderingRollupConfigPath: c.RenderingRollupConfigPath,
		RenderingRPC:              c.RenderingRPC,
		MaxBlocksPerRange:         c.MaxBlocksPerRange,
		ClaimRegistry:             registry,
		EventReplayer:             replayer,
		ReplayMessenger:           messenger,
		Emitters:                  render.NewEmitterSet(extra...),
		RollupConfigHash:          rollupConfigHash,
		DepSetHash:                depSetHash,
		OperatorKey:               key,
		Gas: render.GasPolicy{
			GasLimitExport: c.GasLimitExport,
			GasLimitImport: c.GasLimitImport,
			GasLimitEvent:  c.GasLimitEvent,
			GasLimitClaim:  c.GasLimitClaim,
			GasFeeCap:      new(big.Int).SetUint64(c.GasFeeCap),
			GasTipCap:      new(big.Int).SetUint64(c.GasTipCap),
		},
	}, nil
}

func (c *PrivateInteropCLIConfig) operatorKey() (*ecdsa.PrivateKey, error) {
	if c.OperatorKey == "" {
		return nil, errors.New("private interop: --private-interop.operator-key is required")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(c.OperatorKey, "0x"))
	if err != nil {
		// Deliberately does not echo the value.
		return nil, fmt.Errorf("private interop: --%s is not a valid private key: %w",
			flags.PrivateInteropOperatorKeyFlag.Name, err)
	}
	return key, nil
}

// parseAddress accepts only a full 20-byte hex address, and never the zero address.
//
// common.HexToAddress is silently lenient — it truncates and zero-pads anything — so a typo would
// otherwise become a plausible-looking address that no contract sits at.
func parseAddress(flag, value string) (common.Address, error) {
	if value == "" {
		return common.Address{}, fmt.Errorf("private interop: --%s is required", flag)
	}
	if !common.IsHexAddress(value) {
		return common.Address{}, fmt.Errorf("private interop: --%s is not an address: %q", flag, value)
	}
	addr := common.HexToAddress(value)
	if addr == (common.Address{}) {
		return common.Address{}, fmt.Errorf("private interop: --%s is the zero address", flag)
	}
	return addr, nil
}

// parseHash accepts only a full 32-byte hex hash, and never the zero hash. A zero rollupConfigHash
// or depSetHash would be a claim that commits to nothing about which chain it speaks for.
func parseHash(flag, value string) (common.Hash, error) {
	if value == "" {
		return common.Hash{}, fmt.Errorf("private interop: --%s is required", flag)
	}
	b, err := hexToFixed(value, common.HashLength)
	if err != nil {
		return common.Hash{}, fmt.Errorf("private interop: --%s is not a 32-byte hash: %w", flag, err)
	}
	h := common.BytesToHash(b)
	if h == (common.Hash{}) {
		return common.Hash{}, fmt.Errorf("private interop: --%s is the zero hash", flag)
	}
	return h, nil
}

func hexToFixed(value string, n int) ([]byte, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		return nil, err
	}
	if len(b) != n {
		return nil, fmt.Errorf("%d bytes, want %d", len(b), n)
	}
	return b, nil
}
