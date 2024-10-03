package state

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
)

var emptyAddress common.Address

type Intent struct {
	L1ChainID uint64 `json:"l1ChainID" toml:"l1ChainID"`

	SuperchainRoles SuperchainRoles `json:"superchainRoles" toml:"superchainRoles"`

	FundDevAccounts bool `json:"fundDevAccounts" toml:"fundDevAccounts"`

	ContractArtifactsURL *ArtifactsURL `json:"contractArtifactsURL" toml:"contractArtifactsURL"`

	ContractsRelease string `json:"contractsRelease" toml:"contractsRelease"`

	Chains []*ChainIntent `json:"chains" toml:"chains"`

	GlobalDeployOverrides map[string]any `json:"globalDeployOverrides" toml:"globalDeployOverrides"`
}

func (c *Intent) L1ChainIDBig() *big.Int {
	return big.NewInt(int64(c.L1ChainID))
}

func (c *Intent) Check() error {
	if c.L1ChainID == 0 {
		return fmt.Errorf("l1ChainID must be set")
	}

	if c.ContractsRelease == "dev" {
		return c.checkDev()
	}

	return c.checkProd()
}

func (c *Intent) Chain(id common.Hash) (*ChainIntent, error) {
	for i := range c.Chains {
		if c.Chains[i].ID == id {
			return c.Chains[i], nil
		}
	}

	return nil, fmt.Errorf("chain %d not found", id)
}

func (c *Intent) WriteToFile(path string) error {
	return jsonutil.WriteTOML(c, ioutil.ToAtomicFile(path, 0o755))
}

func (c *Intent) checkDev() error {
	if c.SuperchainRoles.ProxyAdminOwner == emptyAddress {
		return fmt.Errorf("proxyAdminOwner must be set")
	}

	if c.SuperchainRoles.ProtocolVersionsOwner == emptyAddress {
		c.SuperchainRoles.ProtocolVersionsOwner = c.SuperchainRoles.ProxyAdminOwner
	}

	if c.SuperchainRoles.Guardian == emptyAddress {
		c.SuperchainRoles.Guardian = c.SuperchainRoles.ProxyAdminOwner
	}

	if c.ContractArtifactsURL == nil {
		return fmt.Errorf("contractArtifactsURL must be set in dev mode")
	}

	return nil
}

func (c *Intent) checkProd() error {
	if !strings.HasPrefix(c.ContractsRelease, "op-contracts/") {
		return fmt.Errorf("contractsVersion must be either the literal \"dev\" or start with \"op-contracts/\"")
	}

	return nil
}

type SuperchainRoles struct {
	ProxyAdminOwner common.Address `json:"proxyAdminOwner" toml:"proxyAdminOwner"`

	ProtocolVersionsOwner common.Address `json:"protocolVersionsOwner" toml:"protocolVersionsOwner"`

	Guardian common.Address `json:"guardian" toml:"guardian"`
}

type ChainIntent struct {
	ID common.Hash `json:"id" toml:"id"`

	Roles ChainRoles `json:"roles" toml:"roles"`

	DeployOverrides map[string]any `json:"deployOverrides" toml:"deployOverrides"`
}

type ChainRoles struct {
	ProxyAdminOwner common.Address `json:"proxyAdminOwner" toml:"proxyAdminOwner"`

	SystemConfigOwner common.Address `json:"systemConfigOwner" toml:"systemConfigOwner"`

	GovernanceTokenOwner common.Address `json:"governanceTokenOwner" toml:"governanceTokenOwner"`

	UnsafeBlockSigner common.Address `json:"unsafeBlockSigner" toml:"unsafeBlockSigner"`

	Batcher common.Address `json:"batcher" toml:"batcher"`

	Proposer common.Address `json:"proposer" toml:"proposer"`

	Challenger common.Address `json:"challenger" toml:"challenger"`
}

func (c *ChainIntent) Check() error {
	var emptyHash common.Hash
	if c.ID == emptyHash {
		return fmt.Errorf("id must be set")
	}

	if c.Roles.ProxyAdminOwner == emptyAddress {
		return fmt.Errorf("proxyAdminOwner must be set")
	}

	if c.Roles.SystemConfigOwner == emptyAddress {
		c.Roles.SystemConfigOwner = c.Roles.ProxyAdminOwner
	}

	if c.Roles.GovernanceTokenOwner == emptyAddress {
		c.Roles.GovernanceTokenOwner = c.Roles.ProxyAdminOwner
	}

	if c.Roles.UnsafeBlockSigner == emptyAddress {
		return fmt.Errorf("unsafeBlockSigner must be set")
	}

	if c.Roles.Batcher == emptyAddress {
		return fmt.Errorf("batcher must be set")
	}

	return nil
}
