package superchain

import (
	"fmt"
	"path"
	"slices"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

// legacySuperchainKeys are keys still present in the embedded superchain.toml files that the
// Superchain struct deliberately no longer models. They are tolerated by the strict decoder so a
// genuinely new/unexpected key is still rejected while this known legacy field is not.
var legacySuperchainKeys = []string{"protocol_versions_addr"}

func isLegacySuperchainKey(key string) bool {
	return slices.Contains(legacySuperchainKeys, key)
}

type Superchain struct {
	Name                   string         `toml:"name"`
	SuperchainConfigAddr   common.Address `toml:"superchain_config_addr"`
	OpContractsManagerAddr common.Address `toml:"op_contracts_manager_addr"`
	SaferSafesAddr         common.Address `toml:"safer_safes_addr"`
	Hardforks              HardforkConfig
	L1                     L1Config
}

type L1Config struct {
	ChainID   uint64 `toml:"chain_id"`
	PublicRPC string `toml:"public_rpc"`
	Explorer  string `toml:"explorer"`
}

func GetSuperchain(network string) (Superchain, error) {
	return BuiltInConfigs.GetSuperchain(network)
}

func (c *ChainConfigLoader) GetSuperchain(network string) (Superchain, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	var sc Superchain
	if sc, ok := c.superchainsByNetwork[network]; ok {
		return sc, nil
	}

	zr, err := c.configDataReader.Open(path.Join("configs", network, "superchain.toml"))
	if err != nil {
		return sc, err
	}

	if err := jsonutil.DecodeTOMLStrict(zr, &sc, isLegacySuperchainKey); err != nil {
		return sc, fmt.Errorf("error decoding superchain config: %w", err)
	}

	c.superchainsByNetwork[network] = sc
	return sc, nil
}
