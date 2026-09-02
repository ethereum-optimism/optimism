package claimfollow

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
)

// Private Interop's supernode flag group.
//
// The ratified topology (op-private-interop/docs/DESIGN.md, "The supernode follow module") has no
// follower binary: the public supernode serves the private chain's follow endpoint from public
// data. There is no marker in any rollup config: --private-interop.chain-id names the chain and
// --private-interop.genesis supplies the private-chain genesis the supernode projects, and setting
// them is the only activation signal.

// DefaultRoute is the sub-route the module is served at, under the chain's own route:
// `<base>/<chainID>/claimed`. A DISTINCT route rather than a flip of the existing one, so that a
// mispointed consumer fails loudly instead of receiving the public projection's own refs — which a
// sequencing LightCL would force-reset onto.
const DefaultRoute = "claimed"

var (
	ChainIDFlag = &cli.Uint64Flag{
		Name: "private-interop.chain-id",
		Usage: "Chain ID of the private interop chain. The supernode replaces that chain's rollup " +
			"config with the public projection derived from --private-interop.genesis, and serves the " +
			"claimed private view at the chain's `claimed` sibling route.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_CHAIN_ID"),
	}
	GenesisPathFlag = &cli.PathFlag{
		Name: "private-interop.genesis",
		Usage: "Path to the private-chain genesis. The supernode derives both the private genesis " +
			"hash and the public-projection rollup configuration from this local artifact.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_GENESIS"),
	}
	ScanStartBlockFlag = &cli.Uint64Flag{
		Name: "private-interop.claim-scan-start-block",
		Usage: "First public-projection block to scan for claims. Zero scans from genesis, which is correct " +
			"but walks the whole chain; an operator enabling the module against an existing projection " +
			"sets it to the block the registry's first claim landed in.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_CLAIM_SCAN_START_BLOCK"),
	}
)

// Flags is the whole group, registered with the supernode's activity flag set.
var Flags = []cli.Flag{
	ChainIDFlag,
	GenesisPathFlag,
	ScanStartBlockFlag,
}

func init() {
	flags.RegisterActivityFlags(Flags...)
}

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(flags.EnvVarPrefix, name)
}

// CLIConfig is the --private-interop.* group as read from the CLI.
//
// The group is all-or-nothing: Enabled reports whether either activating flag is set, and Check
// then demands both.
type CLIConfig struct {
	ChainID        uint64
	ChainIDSet     bool
	GenesisPath    string
	ScanStartBlock uint64
}

// Enabled reports whether the operator asked for the module at all.
func (c CLIConfig) Enabled() bool {
	return c.ChainIDSet || c.GenesisPath != ""
}

// ReadCLIConfig parses the flag group. A nil context returns the zero-value configuration, which is
// what an embedder that never built a CLI (a test, or a library caller) should get.
func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	if ctx == nil {
		return CLIConfig{}
	}
	return CLIConfig{
		ChainID:        ctx.Uint64(ChainIDFlag.Name),
		ChainIDSet:     ctx.IsSet(ChainIDFlag.Name),
		GenesisPath:    ctx.Path(GenesisPathFlag.Name),
		ScanStartBlock: ctx.Uint64(ScanStartBlockFlag.Name),
	}
}

// Check validates the group once Enabled.
func (c CLIConfig) Check() error {
	if !c.ChainIDSet {
		return fmt.Errorf("%s is required: it names the chain whose rollup config the supernode "+
			"replaces with the public projection", ChainIDFlag.Name)
	}
	if c.GenesisPath == "" {
		return fmt.Errorf("%s is required: the module serves the private chain's "+
			"genesis ref until the first claim lands, and a module that serves nothing deadlocks the "+
			"batcher that would have produced that claim", GenesisPathFlag.Name)
	}
	return nil
}

// Resolve returns the typed configuration. It must not be called before Check has passed.
func (c CLIConfig) Resolve() (Config, error) {
	if err := c.Check(); err != nil {
		return Config{}, err
	}
	genesis, err := c.LoadPrivateChainGenesis()
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		Registry:    predeploys.ClaimRegistryAddr,
		GenesisHash: genesis.ToBlock().Hash(),
		StartBlock:  c.ScanStartBlock,
	}
	return cfg, nil
}

// LoadPrivateChainGenesis reads the operator-supplied local genesis artifact. File access is kept
// outside the pure projection function; callers may load once and project as many times as needed.
func (c CLIConfig) LoadPrivateChainGenesis() (*core.Genesis, error) {
	if c.GenesisPath == "" {
		return nil, fmt.Errorf("%s is required", GenesisPathFlag.Name)
	}
	data, err := os.ReadFile(c.GenesisPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s %q: %w", GenesisPathFlag.Name, c.GenesisPath, err)
	}
	var genesis core.Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		return nil, fmt.Errorf("decoding %s %q: %w", GenesisPathFlag.Name, c.GenesisPath, err)
	}
	return &genesis, nil
}
