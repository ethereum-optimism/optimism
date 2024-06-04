package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"

	opnode "github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-program/host/flags"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

var (
	ErrMissingRollupConfig = errors.New("missing rollup config")
	ErrMissingL2Genesis    = errors.New("missing l2 genesis")
	ErrInvalidL1Head       = errors.New("invalid l1 head")
	ErrInvalidL2Head       = errors.New("invalid l2 head")
	ErrInvalidL2OutputRoot = errors.New("invalid l2 output root")
	ErrL1AndL2Inconsistent = errors.New("l1 and l2 options must be specified together or both omitted")
	ErrInvalidL2Claim      = errors.New("invalid l2 claim")
	ErrInvalidL2ClaimBlock = errors.New("invalid l2 claim block number")
	ErrDataDirRequired     = errors.New("datadir must be specified when in non-fetching mode")
	ErrNoExecInServerMode  = errors.New("exec command must not be set when in server mode")
)

type Config struct {
	Rollup *rollup.Config
	// DataDir is the directory to read/write pre-image data from/to.
	// If not set, an in-memory key-value store is used and fetching data must be enabled
	DataDir string

	// L1Head is the block hash of the L1 chain head block
	L1Head      common.Hash
	L1URL       string
	L1BeaconURL string
	L1TrustRPC  bool
	L1RPCKind   sources.RPCProviderKind

	// L2Head is the l2 block hash contained in the L2 Output referenced by the L2OutputRoot
	L2Head common.Hash
	// L2OutputRoot is the agreed L2 output root to start derivation from
	L2OutputRoot common.Hash
	L2URL        string
	// L2Claim is the claimed L2 output root to verify
	L2Claim common.Hash
	// L2ClaimBlockNumber is the block number the claimed L2 output root is from
	// Must be above 0 and to be a valid claim needs to be above the L2Head block.
	L2ClaimBlockNumber uint64
	// L2ChainConfig is the op-geth chain config for the L2 execution engine
	L2ChainConfig *params.ChainConfig
	// ExecCmd specifies the client program to execute in a separate process.
	// If unset, the fault proof client is run in the same process.
	ExecCmd string

	// ServerMode indicates that the program should run in pre-image server mode and wait for requests.
	// No client program is run.
	ServerMode bool

	// IsCustomChainConfig indicates that the program uses a custom chain configuration
	IsCustomChainConfig bool
}

func (c *Config) Check() error {
	if c.Rollup == nil {
		return ErrMissingRollupConfig
	}
	if err := c.Rollup.Check(); err != nil {
		return err
	}
	if c.L1Head == (common.Hash{}) {
		return ErrInvalidL1Head
	}
	if c.L2Head == (common.Hash{}) {
		return ErrInvalidL2Head
	}
	if c.L2OutputRoot == (common.Hash{}) {
		return ErrInvalidL2OutputRoot
	}
	if c.L2ClaimBlockNumber == 0 {
		return ErrInvalidL2ClaimBlock
	}
	if c.L2ChainConfig == nil {
		return ErrMissingL2Genesis
	}
	if (c.L1URL != "") != (c.L2URL != "") {
		return ErrL1AndL2Inconsistent
	}
	if !c.FetchingEnabled() && c.DataDir == "" {
		return ErrDataDirRequired
	}
	if c.ServerMode && c.ExecCmd != "" {
		return ErrNoExecInServerMode
	}
	return nil
}

func (c *Config) FetchingEnabled() bool {
	// TODO: Include Beacon URL once cancun is active on all chains we fault prove.
	return c.L1URL != "" && c.L2URL != ""
}

// NewConfig creates a Config with all optional values set to the CLI default value
func NewConfig(
	rollupCfg *rollup.Config,
	l2Genesis *params.ChainConfig,
	l1Head common.Hash,
	l2Head common.Hash,
	l2OutputRoot common.Hash,
	l2Claim common.Hash,
	l2ClaimBlockNum uint64,
) *Config {
	_, err := params.LoadOPStackChainConfig(l2Genesis.ChainID.Uint64())
	isCustomConfig := err != nil
	return &Config{
		Rollup:              rollupCfg,
		L2ChainConfig:       l2Genesis,
		L1Head:              l1Head,
		L2Head:              l2Head,
		L2OutputRoot:        l2OutputRoot,
		L2Claim:             l2Claim,
		L2ClaimBlockNumber:  l2ClaimBlockNum,
		L1RPCKind:           sources.RPCKindStandard,
		IsCustomChainConfig: isCustomConfig,
	}
}

func NewConfigFromCLI(log log.Logger, ctx *cli.Context) (*Config, error) {
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}
	rollupCfg, err := opnode.NewRollupConfigFromCLI(log, ctx)
	if err != nil {
		return nil, err
	}
	l2Head := common.HexToHash(ctx.String(flags.L2Head.Name))
	if l2Head == (common.Hash{}) {
		return nil, ErrInvalidL2Head
	}
	l2OutputRoot := common.HexToHash(ctx.String(flags.L2OutputRoot.Name))
	if l2OutputRoot == (common.Hash{}) {
		return nil, ErrInvalidL2OutputRoot
	}
	strClaim := ctx.String(flags.L2Claim.Name)
	l2Claim := common.HexToHash(strClaim)
	// Require a valid hash, with the zero hash explicitly allowed.
	if l2Claim == (common.Hash{}) &&
		strClaim != "0x0000000000000000000000000000000000000000000000000000000000000000" &&
		strClaim != "0000000000000000000000000000000000000000000000000000000000000000" {
		return nil, fmt.Errorf("%w: %v", ErrInvalidL2Claim, strClaim)
	}
	l2ClaimBlockNum := ctx.Uint64(flags.L2BlockNumber.Name)
	l1Head := common.HexToHash(ctx.String(flags.L1Head.Name))
	if l1Head == (common.Hash{}) {
		return nil, ErrInvalidL1Head
	}
	l2GenesisPath := ctx.String(flags.L2GenesisPath.Name)
	var l2ChainConfig *params.ChainConfig
	var isCustomConfig bool
	if l2GenesisPath == "" {
		networkName := ctx.String(flags.Network.Name)
		ch := chaincfg.ChainByName(networkName)
		if ch == nil {
			return nil, fmt.Errorf("flag %s is required for network %s", flags.L2GenesisPath.Name, networkName)
		}
		cfg, err := params.LoadOPStackChainConfig(ch.ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to load chain config for chain %d: %w", ch.ChainID, err)
		}
		l2ChainConfig = cfg
	} else {
		l2ChainConfig, err = loadChainConfigFromGenesis(l2GenesisPath)
		isCustomConfig = true
	}
	if err != nil {
		return nil, fmt.Errorf("invalid genesis: %w", err)
	}
	return &Config{
		Rollup:              rollupCfg,
		DataDir:             ctx.String(flags.DataDir.Name),
		L2URL:               ctx.String(flags.L2NodeAddr.Name),
		L2ChainConfig:       l2ChainConfig,
		L2Head:              l2Head,
		L2OutputRoot:        l2OutputRoot,
		L2Claim:             l2Claim,
		L2ClaimBlockNumber:  l2ClaimBlockNum,
		L1Head:              l1Head,
		L1URL:               ctx.String(flags.L1NodeAddr.Name),
		L1BeaconURL:         ctx.String(flags.L1BeaconAddr.Name),
		L1TrustRPC:          ctx.Bool(flags.L1TrustRPC.Name),
		L1RPCKind:           sources.RPCProviderKind(ctx.String(flags.L1RPCProviderKind.Name)),
		ExecCmd:             ctx.String(flags.Exec.Name),
		ServerMode:          ctx.Bool(flags.Server.Name),
		IsCustomChainConfig: isCustomConfig,
	}, nil
}

func loadChainConfigFromGenesis(path string) (*params.ChainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read l2 genesis file: %w", err)
	}
	var genesis core.Genesis
	err = json.Unmarshal(data, &genesis)
	if err != nil {
		return nil, fmt.Errorf("parse l2 genesis file: %w", err)
	}
	return genesis.Config, nil
}
