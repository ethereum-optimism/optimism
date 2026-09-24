package batcher

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	// ProofCommand enables native execution before mock publication on fresh test deployments.
	ProofCommand string
	// PrivateChainGenesisPath is the private-chain genesis projected by this process: a local path
	// or an http(s) URL.
	PrivateChainGenesisPath string
	// PublicProjectionRPC is the execution client following the public projection.
	PublicProjectionRPC       string
	PublicProjectionRollupRPC string

	// MaxBlocksPerRange is the cadence.
	MaxBlocksPerRange uint64
	// MaxRangeBytes is the uncompressed producer-side byte budget.
	MaxRangeBytes uint64

	// ExtraEmitters are additional application emitters replayed onto the projection, as hex.
	ExtraEmitters []string

	// RollupConfigHash and DepSetHash are optional cross-checks of the claim's two configuration
	// commitments, as hex. The batcher always derives both from the deployed projection config
	// (projection.ConfigHash and private_projection.dependency_set_hash); a value given here must
	// equal the derived one or startup fails.
	RollupConfigHash string
	DepSetHash       string

	// PrivateRollupConfigPath and L1ChainConfigPath are the two pinned artifacts the projection's
	// private_config_hash covers, byte for byte. Required for sp1-private-projection-v1.
	PrivateRollupConfigPath string
	L1ChainConfigPath       string
	// SP1Prover is the producer's prover for sp1-private-projection-v1: network (the default when
	// empty), cpu, mock or native-mock.
	SP1Prover string
	// ProofTimeout and ProofTimeoutPerBlock give the producer timeout
	// ProofTimeout + ProofTimeoutPerBlock × (lastBlock − anchorBlock). Zero takes the default.
	ProofTimeout         time.Duration
	ProofTimeoutPerBlock time.Duration

	// TestHooks are programmatic test hooks. There is no flag for them.
	TestHooks *PrivateInteropTestHooks

	GasLimitExport uint64
	GasLimitImport uint64
	GasLimitEvent  uint64
	GasLimitClaim  uint64
}

// ReadPrivateInteropCLIConfig parses the flag group.
func ReadPrivateInteropCLIConfig(ctx *cli.Context) PrivateInteropCLIConfig {
	return PrivateInteropCLIConfig{
		ProofCommand:              ctx.String(flags.PrivateInteropProofCommandFlag.Name),
		PrivateChainGenesisPath:   ctx.String(flags.PrivateInteropGenesisFlag.Name),
		PublicProjectionRPC:       ctx.String(flags.PrivateInteropPublicProjectionRPCFlag.Name),
		PublicProjectionRollupRPC: ctx.String(flags.PrivateInteropPublicProjectionRollupRPCFlag.Name),
		MaxBlocksPerRange:         ctx.Uint64(flags.PrivateInteropMaxBlocksPerRangeFlag.Name),
		MaxRangeBytes:             ctx.Uint64(flags.PrivateInteropMaxRangeBytesFlag.Name),
		ExtraEmitters:             ctx.StringSlice(flags.PrivateInteropExtraEmittersFlag.Name),
		RollupConfigHash:          ctx.String(flags.PrivateInteropRollupConfigHashFlag.Name),
		DepSetHash:                ctx.String(flags.PrivateInteropDepSetHashFlag.Name),
		GasLimitExport:            ctx.Uint64(flags.PrivateInteropGasLimitExportFlag.Name),
		GasLimitImport:            ctx.Uint64(flags.PrivateInteropGasLimitImportFlag.Name),
		GasLimitEvent:             ctx.Uint64(flags.PrivateInteropGasLimitEventFlag.Name),
		GasLimitClaim:             ctx.Uint64(flags.PrivateInteropGasLimitClaimFlag.Name),
		PrivateRollupConfigPath:   ctx.Path(flags.PrivateInteropPrivateRollupConfigFlag.Name),
		L1ChainConfigPath:         ctx.Path(flags.PrivateInteropL1ChainConfigFlag.Name),
		SP1Prover:                 ctx.String(flags.PrivateInteropSP1ProverFlag.Name),
		ProofTimeout:              ctx.Duration(flags.PrivateInteropProofTimeoutFlag.Name),
		ProofTimeoutPerBlock:      ctx.Duration(flags.PrivateInteropProofTimeoutPerBlockFlag.Name),
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
	if c.PublicProjectionRollupRPC == "" {
		return errors.New("private interop: --private-interop.public-projection-rollup-rpc is required")
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
	if (c.PrivateRollupConfigPath == "") != (c.L1ChainConfigPath == "") {
		return fmt.Errorf("private interop: --%s and --%s are given together",
			flags.PrivateInteropPrivateRollupConfigFlag.Name, flags.PrivateInteropL1ChainConfigFlag.Name)
	}
	if _, err := sp1ProverOrDefault(c.SP1Prover); err != nil {
		return err
	}
	if c.ProofTimeout < 0 || c.ProofTimeoutPerBlock < 0 {
		return fmt.Errorf("private interop: --%s and --%s cannot be negative",
			flags.PrivateInteropProofTimeoutFlag.Name, flags.PrivateInteropProofTimeoutPerBlockFlag.Name)
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
	ProofCommand              string
	PrivateChainGenesisPath   string
	PublicProjectionRPC       string
	PublicProjectionRollupRPC string
	MaxBlocksPerRange         uint64
	MaxRangeBytes             uint64
	ExtraEmitters             []common.Address

	ClaimRegistry    common.Address
	EventReplayer    common.Address
	ReplayMessenger  common.Address
	RollupConfigHash common.Hash
	DepSetHash       common.Hash

	Gas render.GasPolicy

	PrivateRollupConfigPath string
	L1ChainConfigPath       string
	SP1Prover               string
	ProofTimeout            time.Duration
	ProofTimeoutPerBlock    time.Duration
	TestHooks               *PrivateInteropTestHooks
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
	prover, _ := sp1ProverOrDefault(c.SP1Prover)
	timeout, perBlock := c.ProofTimeout, c.ProofTimeoutPerBlock
	if timeout == 0 {
		timeout = flags.DefaultPrivateInteropProofTimeout
	}
	if perBlock == 0 {
		perBlock = flags.DefaultPrivateInteropProofTimeoutPerBlock
	}
	return &PrivateInteropSettings{
		PrivateRollupConfigPath:   c.PrivateRollupConfigPath,
		L1ChainConfigPath:         c.L1ChainConfigPath,
		SP1Prover:                 prover,
		ProofTimeout:              timeout,
		ProofTimeoutPerBlock:      perBlock,
		TestHooks:                 c.TestHooks,
		ProofCommand:              c.ProofCommand,
		PrivateChainGenesisPath:   c.PrivateChainGenesisPath,
		PublicProjectionRPC:       c.PublicProjectionRPC,
		PublicProjectionRollupRPC: c.PublicProjectionRollupRPC,
		MaxBlocksPerRange:         c.MaxBlocksPerRange,
		MaxRangeBytes:             c.MaxRangeBytes,
		ExtraEmitters:             emitters,
		ClaimRegistry:             predeploys.ClaimRegistryAddr,
		EventReplayer:             predeploys.EventReplayerAddr,
		ReplayMessenger:           predeploys.L2toL2CrossDomainMessengerAddr,
		RollupConfigHash:          rollupConfigHash,
		DepSetHash:                depSetHash,
		Gas: render.GasPolicy{
			GasLimitExport: c.GasLimitExport,
			GasLimitImport: c.GasLimitImport,
			GasLimitEvent:  c.GasLimitEvent,
			GasLimitClaim:  c.GasLimitClaim,
		},
	}, nil
}

// SP1 prover modes of the proof command (spec-sound-profile §G.2). ProverNative is the
// execution-mock-v1 mode and is not a flag value: the batcher selects it from the deployment.
const (
	ProverNetwork    = "network"
	ProverCPU        = "cpu"
	ProverMock       = "mock"
	ProverNativeMock = "native-mock"
	ProverNative     = "native"
)

func sp1ProverOrDefault(v string) (string, error) {
	switch v {
	case "":
		return flags.DefaultPrivateInteropSP1Prover, nil
	case ProverNetwork, ProverCPU, ProverMock, ProverNativeMock:
		return v, nil
	default:
		return "", fmt.Errorf("private interop: --%s is %q, want network, cpu, mock or native-mock",
			flags.PrivateInteropSP1ProverFlag.Name, v)
	}
}

// privateInteropProfileInputs is everything the batcher resolves its proof profile from: the
// deployed projection rollup config (read over RPC) and what this process holds locally.
type privateInteropProfileInputs struct {
	Settings *PrivateInteropSettings
	// PrivateRollup is the private chain's rollup config, from --rollup-rpc.
	PrivateRollup *rollup.Config
	// Local is the projection rollup config this process projected from the private genesis.
	Local *rollup.Config
	// Deployed is the projection rollup config the projection's rollup node serves.
	Deployed *rollup.Config
	// DependencySet is the private chain's dependency set, as its rollup node serves it.
	DependencySet []eth.ChainID
	// PrivateRollupJSON and L1ChainConfigJSON are the pinned artifact bytes, nil when not given.
	PrivateRollupJSON, L1ChainConfigJSON []byte
}

// privateInteropProfile is the resolved proof profile.
type privateInteropProfile struct {
	// Rollup is the local projection config carrying the deployed private_projection.
	Rollup                       *rollup.Config
	RollupConfigHash, DepSetHash common.Hash
	// PrivateConfigJSON and L1ConfigJSON are the bytes the producer request carries.
	PrivateConfigJSON, L1ConfigJSON []byte
	// Prover is the request's prover field; empty when the deployment needs no producer.
	Prover string
}

// resolvePrivateInteropProfile takes private_projection from the DEPLOYED projection config and
// never chooses the verifier itself (§B.3, §G.1). It refuses to start unless:
//   - the locally projected config matches the deployed one field for field, apart from
//     private_projection, and the deployed profile passes CheckChain;
//   - the dependency set the rollup node serves hashes to the deployed dependency_set_hash;
//   - the pinned artifacts hash to the deployed private_config_hash (sp1) and describe the
//     batcher's own private chain;
//   - a mock prover is used only with mock_proofs;
//   - the optional hash flags equal the derived commitments.
func resolvePrivateInteropProfile(in privateInteropProfileInputs) (*privateInteropProfile, error) {
	s := in.Settings
	if in.Deployed == nil || in.Deployed.PrivateProjection == nil {
		return nil, errors.New("private interop: the deployed projection rollup config has no private_projection")
	}
	deployedProfile := *in.Deployed.PrivateProjection
	if err := sameExceptPrivateProjection(in.Local, in.Deployed); err != nil {
		return nil, err
	}
	if in.Local.PrivateProjection != nil && in.Local.PrivateProjection.GenesisOutputRoot != deployedProfile.GenesisOutputRoot {
		return nil, fmt.Errorf("private interop: deployed genesis_output_root %s differs from the private genesis output %s",
			deployedProfile.GenesisOutputRoot, in.Local.PrivateProjection.GenesisOutputRoot)
	}
	if err := deployedProfile.CheckChain(in.Deployed.L2ChainID); err != nil {
		return nil, fmt.Errorf("private interop: deployed private_projection: %w", err)
	}
	out := &privateInteropProfile{}
	cfg := *in.Local
	cfg.PrivateProjection = &deployedProfile
	out.Rollup = &cfg

	out.RollupConfigHash = projection.ConfigHash(&deployedProfile, projection.Context{
		ChainID: cfg.L2ChainID, GenesisNumber: cfg.Genesis.L2.Number, GenesisTime: cfg.Genesis.L2Time,
		BlockTime: cfg.BlockTime, GenesisHash: cfg.Genesis.L2.Hash,
	})
	if s.RollupConfigHash != (common.Hash{}) && s.RollupConfigHash != out.RollupConfigHash {
		return nil, fmt.Errorf("private interop: --%s %s differs from the deployed projection ConfigHash %s",
			flags.PrivateInteropRollupConfigHashFlag.Name, s.RollupConfigHash, out.RollupConfigHash)
	}
	out.DepSetHash = deployedProfile.DependencySetHash
	if got := projection.DependencySetHash(in.DependencySet); got != out.DepSetHash {
		return nil, fmt.Errorf("private interop: the rollup node's dependency set hashes to %s, the deployed dependency_set_hash is %s",
			got, out.DepSetHash)
	}
	if s.DepSetHash != (common.Hash{}) && s.DepSetHash != out.DepSetHash {
		return nil, fmt.Errorf("private interop: --%s %s differs from the deployed dependency_set_hash %s",
			flags.PrivateInteropDepSetHashFlag.Name, s.DepSetHash, out.DepSetHash)
	}

	pinned := in.PrivateRollupJSON != nil || in.L1ChainConfigJSON != nil
	if pinned {
		var privateCfg rollup.Config
		if err := json.Unmarshal(in.PrivateRollupJSON, &privateCfg); err != nil {
			return nil, fmt.Errorf("private interop: --%s: %w", flags.PrivateInteropPrivateRollupConfigFlag.Name, err)
		}
		if privateCfg.L2ChainID == nil || in.PrivateRollup.L2ChainID == nil || privateCfg.L2ChainID.Cmp(in.PrivateRollup.L2ChainID) != 0 ||
			privateCfg.Genesis.L2.Hash != in.PrivateRollup.Genesis.L2.Hash {
			return nil, fmt.Errorf("private interop: --%s is not this batcher's private chain (chain ID %v, genesis %s; want %v, %s)",
				flags.PrivateInteropPrivateRollupConfigFlag.Name, privateCfg.L2ChainID, privateCfg.Genesis.L2.Hash,
				in.PrivateRollup.L2ChainID, in.PrivateRollup.Genesis.L2.Hash)
		}
		out.PrivateConfigJSON, out.L1ConfigJSON = in.PrivateRollupJSON, in.L1ChainConfigJSON
	}

	switch deployedProfile.Verifier {
	case projection.InsecureStub:
		if s.ProofCommand != "" {
			return nil, fmt.Errorf("private interop: the deployed verifier %s takes no --%s", deployedProfile.Verifier,
				flags.PrivateInteropProofCommandFlag.Name)
		}
		return out, nil
	case projection.ExecutionMock:
		out.Prover = ProverNative
		if !pinned {
			// Non-consensus under execution-mock-v1: private_config_hash is zero.
			var err error
			if out.PrivateConfigJSON, err = json.Marshal(in.PrivateRollup); err != nil {
				return nil, err
			}
			if l1 := eth.L1ChainConfigByChainID(eth.ChainIDFromBig(in.PrivateRollup.L1ChainID)); l1 != nil {
				if out.L1ConfigJSON, err = json.Marshal(l1); err != nil {
					return nil, err
				}
			}
		}
	case projection.SP1PrivateProjectionV1:
		if !pinned {
			return nil, fmt.Errorf("private interop: %s requires --%s and --%s", deployedProfile.Verifier,
				flags.PrivateInteropPrivateRollupConfigFlag.Name, flags.PrivateInteropL1ChainConfigFlag.Name)
		}
		if got := projection.PrivateConfigHash(in.PrivateRollupJSON, in.L1ChainConfigJSON); got != deployedProfile.PrivateConfigHash {
			return nil, fmt.Errorf("private interop: the pinned config files hash to %s, the deployed private_config_hash is %s",
				got, deployedProfile.PrivateConfigHash)
		}
		out.Prover = s.SP1Prover
		if (out.Prover == ProverMock || out.Prover == ProverNativeMock) && !deployedProfile.MockProofs {
			return nil, fmt.Errorf("private interop: --%s %s needs mock_proofs in the deployed projection config",
				flags.PrivateInteropSP1ProverFlag.Name, out.Prover)
		}
	default:
		return nil, fmt.Errorf("private interop: unsupported deployed verifier %q", deployedProfile.Verifier)
	}
	if s.ProofCommand == "" {
		return nil, fmt.Errorf("private interop: the deployed verifier %s requires --%s", deployedProfile.Verifier,
			flags.PrivateInteropProofCommandFlag.Name)
	}
	return out, nil
}

// sameExceptPrivateProjection compares two rollup configs field for field, ignoring
// private_projection.
func sameExceptPrivateProjection(local, deployed *rollup.Config) error {
	if local == nil {
		return errors.New("private interop: no locally projected rollup config")
	}
	a, b := *local, *deployed
	a.PrivateProjection, b.PrivateProjection = nil, nil
	aj, err := json.Marshal(&a)
	if err != nil {
		return err
	}
	bj, err := json.Marshal(&b)
	if err != nil {
		return err
	}
	if !bytes.Equal(aj, bj) {
		return errors.New("private interop: the deployed projection rollup config differs from the one projected from the private genesis")
	}
	return nil
}

// parseOptionalHash is parseHash for an optional cross-check: empty is allowed and yields the
// zero hash, meaning "not pinned". A value that IS given must be a full, non-zero 32-byte hash.
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
