package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

func pcdSuperRootFromArtifacts(artifacts []pcdChainArtifacts) (common.Hash, uint64, error) {
	if len(artifacts) == 0 {
		return common.Hash{}, 0, errors.New("compute PCD super root: no chain artifacts")
	}

	chainOutputs := make([]eth.ChainIDAndOutput, 0, len(artifacts))
	var genesisTime uint64
	for i, artifact := range artifacts {
		genesis, err := readPCDGenesis(artifact.genesisPath)
		if err != nil {
			return common.Hash{}, 0, err
		}
		if genesis.Config == nil || genesis.Config.ChainID == nil {
			return common.Hash{}, 0, fmt.Errorf("read PCD genesis %s: missing chain ID", artifact.genesisPath)
		}

		rollupConfig, err := readPCDRollupConfig(artifact.rollupPath)
		if err != nil {
			return common.Hash{}, 0, err
		}
		if rollupConfig.L2ChainID != nil && rollupConfig.L2ChainID.Cmp(genesis.Config.ChainID) != 0 {
			return common.Hash{}, 0, fmt.Errorf(
				"PCD artifact chain ID mismatch: genesis %s has %s, rollup %s has %s",
				artifact.genesisPath,
				genesis.Config.ChainID,
				artifact.rollupPath,
				rollupConfig.L2ChainID,
			)
		}

		header := genesis.ToBlock().Header()
		if header.WithdrawalsHash == nil {
			return common.Hash{}, 0, fmt.Errorf(
				"read PCD genesis %s: block %s has no withdrawals hash",
				artifact.genesisPath,
				header.Hash(),
			)
		}
		if header.Time != rollupConfig.Genesis.L2Time {
			return common.Hash{}, 0, fmt.Errorf(
				"PCD artifact genesis time mismatch: genesis %s has %d, rollup %s has %d",
				artifact.genesisPath,
				header.Time,
				artifact.rollupPath,
				rollupConfig.Genesis.L2Time,
			)
		}
		if i == 0 {
			genesisTime = rollupConfig.Genesis.L2Time
		} else if rollupConfig.Genesis.L2Time != genesisTime {
			return common.Hash{}, 0, fmt.Errorf(
				"PCD rollup genesis time mismatch: %s has %d, expected %d",
				artifact.rollupPath,
				rollupConfig.Genesis.L2Time,
				genesisTime,
			)
		}

		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromBig(genesis.Config.ChainID),
			Output:  pcdOutputRoot(header),
		})
	}

	// The aggregation follows (*SuperRootMigrator).calculateSuperRoot in op-chain-ops/script/check_super_root.go.
	return common.Hash(eth.SuperRoot(eth.NewSuperV1(genesisTime, chainOutputs...))), genesisTime, nil
}

// pcdOutputRoot follows CalculateOutputRoot in op-chain-ops/cmd/check-output-root/main.go.
func pcdOutputRoot(header *types.Header) eth.Bytes32 {
	return eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(header.Root),
		MessagePasserStorageRoot: eth.Bytes32(*header.WithdrawalsHash),
		BlockHash:                header.Hash(),
	})
}

func readPCDGenesis(path string) (*core.Genesis, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read PCD genesis %s: %w", path, err)
	}
	var genesis core.Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		return nil, fmt.Errorf("decode PCD genesis %s: %w", path, err)
	}
	return &genesis, nil
}

func readPCDRollupConfig(path string) (*rollup.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read PCD rollup config %s: %w", path, err)
	}
	var config rollup.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("decode PCD rollup config %s: %w", path, err)
	}
	return &config, nil
}

func pcdOracleArtifactPaths(artifacts []pcdChainArtifacts) []string {
	paths := make([]string, 0, len(artifacts)*2)
	for _, artifact := range artifacts {
		paths = append(paths, artifact.genesisPath, artifact.rollupPath)
	}
	return paths
}
