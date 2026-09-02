package batcher

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
)

// PrivateInteropCLIConfig is the --private-interop.* flag group as read from the CLI.
//
// It holds the flags' raw values and does every conversion in
// Check, so that a malformed address is a startup error naming the flag rather than a zero value
// that only fails once a range is being built. Resolve returns the typed form, and never runs
// unless Check has passed.
//
// The group is validated only after the loaded rollup config declares private interop. It is
// ALL-OR-NOTHING: there is no half-configured Private Interop batcher, because every field below is
// load-bearing for bytes that go on L1.
type PrivateInteropCLIConfig struct {
	// PrivateChainGenesisPath is the private-chain genesis projected by this process: a local path
	// or an http(s) URL.
	PrivateChainGenesisPath string
	// PublicProjectionRPC is the execution client following the public projection.
	PublicProjectionRPC string

	// MaxBlocksPerRange is the cadence.
	MaxBlocksPerRange uint64
	// MaxRangeBytes is the uncompressed producer-side byte budget.
	MaxRangeBytes uint64

	// ExtraEmitters are additional application emitters replayed onto the projection, as hex.
	ExtraEmitters []string

	// RollupConfigHash and DepSetHash are the claim's two configuration commitments, as hex. Empty
	// means derive: keccak256 of the canonical JSON of the projected rollup config, and of the
	// dependency set the rollup node serves.
	RollupConfigHash string
	DepSetHash       string

	GasLimitExport uint64
	GasLimitImport uint64
	GasLimitEvent  uint64
	GasLimitClaim  uint64
}

// ReadPrivateInteropCLIConfig parses the flag group.
func ReadPrivateInteropCLIConfig(ctx *cli.Context) PrivateInteropCLIConfig {
	return PrivateInteropCLIConfig{
		PrivateChainGenesisPath: ctx.String(flags.PrivateInteropGenesisFlag.Name),
		PublicProjectionRPC:     ctx.String(flags.PrivateInteropPublicProjectionRPCFlag.Name),
		MaxBlocksPerRange:       ctx.Uint64(flags.PrivateInteropMaxBlocksPerRangeFlag.Name),
		MaxRangeBytes:           ctx.Uint64(flags.PrivateInteropMaxRangeBytesFlag.Name),
		ExtraEmitters:           ctx.StringSlice(flags.PrivateInteropExtraEmittersFlag.Name),
		RollupConfigHash:        ctx.String(flags.PrivateInteropRollupConfigHashFlag.Name),
		DepSetHash:              ctx.String(flags.PrivateInteropDepSetHashFlag.Name),
		GasLimitExport:          ctx.Uint64(flags.PrivateInteropGasLimitExportFlag.Name),
		GasLimitImport:          ctx.Uint64(flags.PrivateInteropGasLimitImportFlag.Name),
		GasLimitEvent:           ctx.Uint64(flags.PrivateInteropGasLimitEventFlag.Name),
		GasLimitClaim:           ctx.Uint64(flags.PrivateInteropGasLimitClaimFlag.Name),
	}
}

// Enabled reports whether the group is in use. The genesis path is the switch: there is no marker
// in the rollup config, so a batcher is a private-interop batcher exactly when it is given the
// private-chain genesis to project.
func (c *PrivateInteropCLIConfig) Enabled() bool {
	return c.PrivateChainGenesisPath != ""
}

// minProjectionTxGas is the intrinsic gas of a transaction with calldata. A gas limit under it
// cannot pay for the transaction's own existence, so it is a configuration error rather than an
// under-provisioned replay.
const minProjectionTxGas = 21_000

// Check validates the whole group, and is the only place that decides what a valid Private Interop
// configuration is.
//
// Hashes are checked for the zero value as well as syntax. The projection contracts use fixed
// predeploy addresses and therefore are not operator configuration.
func (c *PrivateInteropCLIConfig) Check() error {
	if c.PrivateChainGenesisPath == "" {
		return errors.New("private interop: --private-interop.genesis is required")
	}
	if c.PublicProjectionRPC == "" {
		return errors.New("private interop: --private-interop.public-projection-rpc is required")
	}
	if c.MaxBlocksPerRange == 0 {
		return errors.New("private interop: --private-interop.max-blocks-per-range must be greater than zero")
	}
	if c.MaxRangeBytes == 0 {
		return errors.New("private interop: --private-interop.max-range-bytes must be greater than zero")
	}
	if _, err := parseEmitters(c.ExtraEmitters); err != nil {
		return err
	}
	for _, f := range []struct {
		flag, value string
	}{
		{flags.PrivateInteropRollupConfigHashFlag.Name, c.RollupConfigHash},
		{flags.PrivateInteropDepSetHashFlag.Name, c.DepSetHash},
	} {
		if _, err := parseOptionalHash(f.flag, f.value); err != nil {
			return err
		}
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
		if f.limit < minProjectionTxGas {
			return fmt.Errorf("private interop: --%s is %d, below the %d intrinsic gas a transaction costs",
				f.flag, f.limit, minProjectionTxGas)
		}
	}
	return nil
}

// PrivateInteropSettings is the group in its typed form.
type PrivateInteropSettings struct {
	PrivateChainGenesisPath string
	PublicProjectionRPC     string
	MaxBlocksPerRange       uint64
	MaxRangeBytes           uint64
	ExtraEmitters           []common.Address

	ClaimRegistry    common.Address
	EventReplayer    common.Address
	ReplayMessenger  common.Address
	RollupConfigHash common.Hash
	DepSetHash       common.Hash

	Gas render.GasPolicy
}

// Resolve converts the raw group into its typed form. It re-runs Check first, so a caller cannot
// resolve a configuration nobody validated.
func (c *PrivateInteropCLIConfig) Resolve() (*PrivateInteropSettings, error) {
	if err := c.Check(); err != nil {
		return nil, err
	}
	rollupConfigHash, _ := parseOptionalHash(flags.PrivateInteropRollupConfigHashFlag.Name, c.RollupConfigHash)
	depSetHash, _ := parseOptionalHash(flags.PrivateInteropDepSetHashFlag.Name, c.DepSetHash)
	emitters, _ := parseEmitters(c.ExtraEmitters)
	return &PrivateInteropSettings{
		PrivateChainGenesisPath: c.PrivateChainGenesisPath,
		PublicProjectionRPC:     c.PublicProjectionRPC,
		MaxBlocksPerRange:       c.MaxBlocksPerRange,
		MaxRangeBytes:           c.MaxRangeBytes,
		ExtraEmitters:           emitters,
		ClaimRegistry:           predeploys.ClaimRegistryAddr,
		EventReplayer:           predeploys.EventReplayerAddr,
		ReplayMessenger:         predeploys.L2toL2CrossDomainMessengerAddr,
		RollupConfigHash:        rollupConfigHash,
		DepSetHash:              depSetHash,
		Gas: render.GasPolicy{
			GasLimitExport: c.GasLimitExport,
			GasLimitImport: c.GasLimitImport,
			GasLimitEvent:  c.GasLimitEvent,
			GasLimitClaim:  c.GasLimitClaim,
		},
	}, nil
}

// parseOptionalHash is parseHash for a commitment the batcher can derive itself: empty is allowed
// and yields the zero hash, which the service replaces with the derived value. A value that IS
// given must be a full, non-zero 32-byte hash.
func parseOptionalHash(flag, value string) (common.Hash, error) {
	if value == "" {
		return common.Hash{}, nil
	}
	return parseHash(flag, value)
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

// parseEmitters accepts full 20-byte hex addresses, none zero and none repeated: an emitter set is
// consensus-relevant, and a typo that silently dropped or doubled an emitter would renumber every
// replayed log.
func parseEmitters(values []string) ([]common.Address, error) {
	out := make([]common.Address, 0, len(values))
	seen := make(map[common.Address]struct{}, len(values))
	for _, value := range values {
		b, err := hexToFixed(value, common.AddressLength)
		if err != nil {
			return nil, fmt.Errorf("private interop: --%s value %q: %w", flags.PrivateInteropExtraEmittersFlag.Name, value, err)
		}
		addr := common.BytesToAddress(b)
		if addr == (common.Address{}) {
			return nil, fmt.Errorf("private interop: --%s cannot contain the zero address", flags.PrivateInteropExtraEmittersFlag.Name)
		}
		if _, ok := seen[addr]; ok {
			return nil, fmt.Errorf("private interop: --%s repeats %s", flags.PrivateInteropExtraEmittersFlag.Name, addr)
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
	}
	return out, nil
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
