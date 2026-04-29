package superchain

import (
	"fmt"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/common"
)

type Superchain struct {
	Name                   string         `toml:"name"`
	ProtocolVersionsAddr   common.Address `toml:"protocol_versions_addr"`
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

	if _, err := toml.NewDecoder(zr).Decode(&sc); err != nil {
		return sc, fmt.Errorf("error decoding superchain config: %w", err)
	}

	c.superchainsByNetwork[network] = sc
	return sc, nil
}
