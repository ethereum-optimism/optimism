package state

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
)

var emptyAddress common.Address

type Intent struct {
	L1ChainID uint64 `json:"l1ChainID" toml:"l1ChainID"`

	SuperchainRoles SuperchainRoles `json:"superchainRoles" toml:"-"`

	FundDevAccounts bool `json:"fundDevAccounts" toml:"fundDevAccounts"`

	L1ContractsLocator *opcm.ArtifactsLocator `json:"l1ContractsLocator" toml:"l1ContractsLocator"`

	L2ContractsLocator *opcm.ArtifactsLocator `json:"l2ContractsLocator" toml:"l2ContractsLocator"`

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

	if c.L1ContractsLocator == nil {
		c.L1ContractsLocator = opcm.DefaultL1ContractsLocator
	}

	if c.L2ContractsLocator == nil {
		c.L2ContractsLocator = opcm.DefaultL2ContractsLocator
	}

	var err error
	if c.L1ContractsLocator.IsTag() {
		err = c.checkL1Prod()
	} else {
		err = c.checkL1Dev()
	}
	if err != nil {
		return err
	}

	if c.L2ContractsLocator.IsTag() {
		if err := c.checkL2Prod(); err != nil {
			return err
		}
	}

	return nil
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

func (c *Intent) checkL1Prod() error {
	versions, err := opcm.StandardL1VersionsFor(c.L1ChainID)
	if err != nil {
		return err
	}

	if _, ok := versions.Releases[c.L1ContractsLocator.Tag]; !ok {
		return fmt.Errorf("tag '%s' not found in standard versions", c.L1ContractsLocator.Tag)
	}

	return nil
}

func (c *Intent) checkL1Dev() error {
	if c.SuperchainRoles.ProxyAdminOwner == emptyAddress {
		return fmt.Errorf("proxyAdminOwner must be set")
	}

	if c.SuperchainRoles.ProtocolVersionsOwner == emptyAddress {
		c.SuperchainRoles.ProtocolVersionsOwner = c.SuperchainRoles.ProxyAdminOwner
	}

	if c.SuperchainRoles.Guardian == emptyAddress {
		c.SuperchainRoles.Guardian = c.SuperchainRoles.ProxyAdminOwner
	}

	return nil
}

func (c *Intent) checkL2Prod() error {
	_, err := opcm.StandardArtifactsURLForTag(c.L2ContractsLocator.Tag)
	return err
}

type SuperchainRoles struct {
	ProxyAdminOwner common.Address `json:"proxyAdminOwner" toml:"proxyAdminOwner"`

	ProtocolVersionsOwner common.Address `json:"protocolVersionsOwner" toml:"protocolVersionsOwner"`

	Guardian common.Address `json:"guardian" toml:"guardian"`
}

type ChainIntent struct {
	ID common.Hash `json:"id" toml:"id"`

	BaseFeeVaultRecipient common.Address `json:"baseFeeVaultRecipient" toml:"baseFeeVaultRecipient"`

	L1FeeVaultRecipient common.Address `json:"l1FeeVaultRecipient" toml:"l1FeeVaultRecipient"`

	SequencerFeeVaultRecipient common.Address `json:"sequencerFeeVaultRecipient" toml:"sequencerFeeVaultRecipient"`

	Eip1559Denominator uint64 `json:"eip1559Denominator" toml:"eip1559Denominator"`

	Eip1559Elasticity uint64 `json:"eip1559Elasticity" toml:"eip1559Elasticity"`

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
