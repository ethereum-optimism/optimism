package claimfollow

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
)

// Private Interop's supernode flag group.
//
// The ratified topology (op-private-interop/docs/DESIGN.md, "The supernode follow module") has no
// follower binary: the public supernode serves the private chain's follow endpoint from public
// data. So everything the module needs is a flag on the stock supernode, and the group is INERT
// unless --private-interop.enabled is set — nothing here changes a stock supernode's behaviour by
// existing, which TestDormantByDefault pins.
//
// Enabled, the group is ALL-OR-NOTHING (Check): there is no half-configured module, because a
// module that scanned the wrong chain or the wrong address would serve confident refs about
// nothing.

// DefaultRoute is the sub-route the module is served at, under the chain's own route:
// `<base>/<chainID>/claimed`. A DISTINCT route rather than a flip of the existing one, so that a
// mispointed consumer fails loudly instead of receiving the RENDERING chain's own refs — which a
// sequencing LightCL would force-reset onto.
const DefaultRoute = "claimed"

var (
	EnabledFlag = &cli.BoolFlag{
		Name: "private-interop.enabled",
		Usage: "Enable the Private Interop claim follow module: for one configured chain, serve that " +
			"chain's PRIVATE counterpart's follow endpoint (optimism_syncStatus) at a sibling route, " +
			"computed entirely from this supernode's own derived data. Requires the rest of the " +
			"--private-interop.* group.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_ENABLED"),
	}
	ChainIDFlag = &cli.Uint64Flag{
		Name: "private-interop.chain-id",
		Usage: "Chain ID of the RENDERING chain whose claims are followed. It must be one of --chains: " +
			"the module reads that chain's derived blocks and receipts in process, and holds no " +
			"credentials of its own.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_CHAIN_ID"),
	}
	ClaimRegistryFlag = &cli.StringFlag{
		Name: "private-interop.claim-registry",
		Usage: "Address of the ClaimRegistry on the rendering chain. It is placed by the rendering's " +
			"genesis builder, so it is per-deployment configuration and has no default; a zero " +
			"address fails loudly rather than scanning for claims that can never appear.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_CLAIM_REGISTRY"),
	}
	GenesisHashFlag = &cli.StringFlag{
		Name: "private-interop.genesis-hash",
		Usage: "The PRIVATE chain's genesis block hash. REQUIRED whenever the module is enabled: it " +
			"is the one field of the private genesis ref that no public data carries, and the module " +
			"serves that ref until the first claim lands. Without it the module says nothing, and the " +
			"operator's own batcher -- which shares the op-node this feeds -- will not load a block " +
			"until it is told a non-zero current_l1, so no claim ever comes. Required here so that is " +
			"a startup failure instead of a bootstrap that hangs with every component looking healthy.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_GENESIS_HASH"),
	}
	RouteFlag = &cli.StringFlag{
		Name: "private-interop.route",
		Usage: "Sub-route the follow endpoint is served at, under the chain's own route: " +
			"<base>/<chainID>/<route>. One path segment, no slashes.",
		Value:   DefaultRoute,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_ROUTE"),
	}
	ScanStartBlockFlag = &cli.Uint64Flag{
		Name: "private-interop.claim-scan-start-block",
		Usage: "First rendering block to scan for claims. Zero scans from genesis, which is correct " +
			"but walks the whole chain; an operator enabling the module against an existing rendering " +
			"sets it to the block the registry's first claim landed in.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_CLAIM_SCAN_START_BLOCK"),
	}
)

// Flags is the whole group, registered with the supernode's activity flag set.
var Flags = []cli.Flag{
	EnabledFlag,
	ChainIDFlag,
	ClaimRegistryFlag,
	GenesisHashFlag,
	RouteFlag,
	ScanStartBlockFlag,
}

func init() {
	flags.RegisterActivityFlags(Flags...)
}

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(flags.EnvVarPrefix, name)
}

// isHash32 reports whether s is exactly 32 bytes of hex, with or without the 0x.
//
// common.HexToHash is lenient -- it left-pads anything shorter and silently truncates anything
// longer -- so a typo'd hash would otherwise become a DIFFERENT valid-looking hash, and the module
// would serve a genesis ref for a block that does not exist. That is the failure this whole flag
// exists to avoid, so the length is checked rather than assumed.
func isHash32(s string) bool {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(trimmed) != 2*common.HashLength {
		return false
	}
	_, err := hex.DecodeString(trimmed)
	return err == nil
}

// CLIConfig is the --private-interop.* group as read from the CLI.
//
// It holds the flags' RAW values and does every conversion in Resolve, so that a malformed address
// is a startup error naming the flag rather than a zero value that only fails once the module is
// scanning.
type CLIConfig struct {
	Enabled        bool
	ChainID        uint64
	ClaimRegistry  string
	GenesisHash    string
	Route          string
	ScanStartBlock uint64
}

// ReadCLIConfig parses the flag group. A nil context reads as disabled, which is what an
// embedder that never built a CLI (a test, or a library caller) should get.
func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	if ctx == nil {
		return CLIConfig{}
	}
	return CLIConfig{
		Enabled:        ctx.Bool(EnabledFlag.Name),
		ChainID:        ctx.Uint64(ChainIDFlag.Name),
		ClaimRegistry:  ctx.String(ClaimRegistryFlag.Name),
		GenesisHash:    ctx.String(GenesisHashFlag.Name),
		Route:          ctx.String(RouteFlag.Name),
		ScanStartBlock: ctx.Uint64(ScanStartBlockFlag.Name),
	}
}

// Check validates the group. A disabled group always passes, whatever else is set: the flags are
// inert, and refusing to start over a stale value in an unused group would be a worse failure than
// ignoring it.
func (c CLIConfig) Check() error {
	if !c.Enabled {
		return nil
	}
	if c.ChainID == 0 {
		return fmt.Errorf("%s is required when %s is set", ChainIDFlag.Name, EnabledFlag.Name)
	}
	if c.ClaimRegistry == "" {
		return fmt.Errorf("%s is required when %s is set", ClaimRegistryFlag.Name, EnabledFlag.Name)
	}
	if !common.IsHexAddress(c.ClaimRegistry) {
		return fmt.Errorf("%s is not a hex address: %q", ClaimRegistryFlag.Name, c.ClaimRegistry)
	}
	if common.HexToAddress(c.ClaimRegistry) == (common.Address{}) {
		return fmt.Errorf("%s must not be the zero address", ClaimRegistryFlag.Name)
	}
	if c.GenesisHash == "" {
		return fmt.Errorf("%s is required when %s is set: the module serves the private chain's "+
			"genesis ref until the first claim lands, and a module that serves nothing deadlocks the "+
			"batcher that would have produced that claim", GenesisHashFlag.Name, EnabledFlag.Name)
	}
	if !isHash32(c.GenesisHash) {
		return fmt.Errorf("%s is not a 32-byte hex hash: %q", GenesisHashFlag.Name, c.GenesisHash)
	}
	if common.HexToHash(c.GenesisHash) == (common.Hash{}) {
		return fmt.Errorf("%s must not be the zero hash", GenesisHashFlag.Name)
	}
	route := strings.TrimSpace(c.Route)
	if route == "" {
		return fmt.Errorf("%s must not be empty", RouteFlag.Name)
	}
	if strings.ContainsRune(route, '/') {
		// The shared router keys on the first path segment and hands the remainder to the chain's
		// handler, so a route with a slash in it would mount somewhere nobody configured.
		return fmt.Errorf("%s must be a single path segment with no '/': %q", RouteFlag.Name, route)
	}
	return nil
}

// Resolve returns the typed configuration. It must not be called before Check has passed.
func (c CLIConfig) Resolve() (eth.ChainID, Config, string, error) {
	if err := c.Check(); err != nil {
		return eth.ChainID{}, Config{}, "", err
	}
	if !c.Enabled {
		return eth.ChainID{}, Config{}, "", errors.New("the private-interop group is not enabled")
	}
	cfg := Config{
		Registry:    common.HexToAddress(c.ClaimRegistry),
		GenesisHash: common.HexToHash(c.GenesisHash),
		StartBlock:  c.ScanStartBlock,
	}
	return eth.ChainIDFromUInt64(c.ChainID), cfg, "/" + strings.TrimSpace(c.Route), nil
}
