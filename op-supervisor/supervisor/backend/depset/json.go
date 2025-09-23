package depset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	_ DependencySetSource   = (*JSONDependencySetLoader)(nil)
	_ RollupConfigSetSource = (*JSONRollupConfigSetLoader)(nil)
	_ RollupConfigSetSource = (*JSONRollupConfigsLoader)(nil)
)

// JSONDependencySetLoader loads a dependency set from a file-path.
type JSONDependencySetLoader struct {
	Path string
}

func (j *JSONDependencySetLoader) LoadDependencySet(ctx context.Context) (DependencySet, error) {
	f, err := os.Open(j.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open dependency set: %w", err)
	}
	defer f.Close()
	return ParseJSONDependencySet(f)
}

func ParseJSONDependencySet(f io.Reader) (DependencySet, error) {
	dec := json.NewDecoder(f)
	var out StaticConfigDependencySet
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode dependency set: %w", err)
	}
	return &out, nil
}

type JSONRollupConfigSetLoader struct {
	Path string
}

func (j *JSONRollupConfigSetLoader) LoadRollupConfigSet(ctx context.Context) (RollupConfigSet, error) {
	f, err := os.Open(j.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open rollup config set: %w", err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var out StaticRollupConfigSet
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode rollup config set: %w", err)
	}
	return &out, nil
}

// JSONRollupConfigsLoader loads a set of op-node rollup.json configs into
// a supervisor rollup config set.
// The [PathPattern] is a glob pattern that matches the rollup.json files, e.g.
// "configs/rollup-*.json". See https://pkg.go.dev/path/filepath#Glob for more details.
// Because the [rollup.Config] doesn't include the genesis L1 block timestamp, they are
// queried from the L1 RPC at URL [L1RPCURL].
type JSONRollupConfigsLoader struct {
	PathPattern string
	L1RPCURL    string
}

func (j *JSONRollupConfigsLoader) LoadRollupConfigSet(ctx context.Context) (RollupConfigSet, error) {
	client, err := ethclient.Dial(j.L1RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()
	return j.loadRollupConfigSet(ctx, client)
}

type headerByHashClient interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
}

// loadRollupConfigSet splits out the core logic for better testing.
func (j *JSONRollupConfigsLoader) loadRollupConfigSet(ctx context.Context, client headerByHashClient) (RollupConfigSet, error) {
	matches, err := filepath.Glob(j.PathPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob files: %w", err)
	}

	configs := make(map[eth.ChainID]*StaticRollupConfig)

	for _, path := range matches {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open rollup config %s: %w", path, err)
		}
		defer file.Close()

		var cfg rollup.Config
		if err = cfg.ParseRollupConfig(file); err != nil {
			return nil, fmt.Errorf("failed to parse rollup config %s: %w", path, err)
		}
		chainID := eth.ChainIDFromBig(cfg.L2ChainID)

		l1Genesis, err := client.HeaderByHash(ctx, cfg.Genesis.L1.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get L1 genesis header for hash %s (chainID: %s): %w", cfg.Genesis.L1.Hash, chainID, err)
		}

		configs[chainID] = StaticRollupConfigFromRollupConfig(&cfg, l1Genesis.Time)
	}

	return StaticRollupConfigSet(configs), nil
}
