package interopgen

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
)

type L1Config struct {
	ChainID *big.Int
	genesis.DevL1DeployConfig
	Prefund map[common.Address]*big.Int
}

func (c *L1Config) Check(log log.Logger) error {
	if c.ChainID == nil {
		return errors.New("missing L1 chain ID")
	}
	// nothing to check on c.DevL1DeployConfig
	return nil
}

type SuperchainConfig struct {
	Deployer common.Address

	FinalSystemOwner common.Address
	ProxyAdminOwner  common.Address

	genesis.SuperchainL1DeployConfig
}

func (c *SuperchainConfig) Check(log log.Logger) error {
	if c.Deployer == (common.Address{}) {
		return errors.New("missing superchain deployer address")
	}
	if c.ProxyAdminOwner == (common.Address{}) {
		return errors.New("missing superchain ProxyAdminOwner address")
	}
	if err := c.SuperchainL1DeployConfig.Check(log); err != nil {
		return err
	}
	return nil
}

type L2Config struct {
	Deployer common.Address // account used to deploy contracts to L2
	genesis.L2InitializationConfig
	genesis.FaultProofDeployConfig
	Prefund map[common.Address]*big.Int
}

func (c *L2Config) Check(log log.Logger) error {
	if c.Deployer == (common.Address{}) {
		return errors.New("missing L2 deployer address")
	}
	if err := c.L2InitializationConfig.Check(log); err != nil {
		return err
	}
	if !c.FaultProofDeployConfig.UseFaultProofs {
		return errors.New("must set UseFaultProofs: legacy output oracle is not supported")
	}
	if err := c.FaultProofDeployConfig.Check(log); err != nil {
		return err
	}
	return nil
}

type WorldConfig struct {
	L1         *L1Config
	Superchain *SuperchainConfig
	L2s        map[string]*L2Config
}

func (c *WorldConfig) Check(log log.Logger) error {
	if err := c.L1.Check(log); err != nil {
		return fmt.Errorf("invalid L1 config: %w", err)
	}
	if err := c.Superchain.Check(log); err != nil {
		return fmt.Errorf("invalid Superchain config: %w", err)
	}
	for l2ChainID, l2Cfg := range c.L2s {
		if err := l2Cfg.Check(log.New("l2", &l2ChainID)); err != nil {
			return fmt.Errorf("invalid L2 (chain ID %s) config: %w", l2ChainID, err)
		}
	}
	return nil
}
