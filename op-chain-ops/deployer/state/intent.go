package state

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
)

var emptyAddress common.Address

type Intent struct {
	L1ChainID uint64 `json:"l1ChainID" toml:"l1ChainID"`

	SuperchainRoles SuperchainRoles `json:"superchainRoles" toml:"superchainRoles"`

	UseFaultProofs bool `json:"useFaultProofs" toml:"useFaultProofs"`

	UseAltDA bool `json:"useAltDA" toml:"useAltDA"`

	FundDevAccounts bool `json:"fundDevAccounts" toml:"fundDevAccounts"`

	ContractArtifactsURL *ArtifactsURL `json:"contractArtifactsURL" toml:"contractArtifactsURL"`

	Chains []Chain `json:"chains" toml:"chains"`
}

func (c Intent) L1ChainIDBig() *big.Int {
	return big.NewInt(int64(c.L1ChainID))
}
func (c Intent) Check() error {
	if c.L1ChainID == 0 {
		return fmt.Errorf("l1ChainID must be set")
	}

	if c.UseFaultProofs && c.UseAltDA {
		return fmt.Errorf("cannot use both fault proofs and alt-DA")
	}

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
		return fmt.Errorf("contractArtifactsURL must be set")
	}

	if c.ContractArtifactsURL.Scheme != "file" {
		return fmt.Errorf("contractArtifactsURL must be a file URL")
	}

	return nil
}

func (c Intent) Chain(id uint64) (Chain, error) {
	for i := range c.Chains {
		if c.Chains[i].ID == id {
			return c.Chains[i], nil
		}
	}

	return Chain{}, fmt.Errorf("chain %d not found", id)
}

func (c Intent) WriteToFile(path string) error {
	return jsonutil.WriteTOML(c, ioutil.ToAtomicFile(path, 0o755))
}

type SuperchainRoles struct {
	ProxyAdminOwner common.Address `json:"proxyAdminOwner" toml:"proxyAdminOwner"`

	ProtocolVersionsOwner common.Address `json:"protocolVersionsOwner" toml:"protocolVersionsOwner"`

	Guardian common.Address `json:"guardian" toml:"guardian"`
}

type Chain struct {
	ID uint64 `json:"id"`

	Roles ChainRoles `json:"roles"`

	Overrides map[string]any `json:"overrides"`
}

type ChainRoles struct {
	ProxyAdminOwner common.Address `json:"proxyAdminOwner"`

	SystemConfigOwner common.Address `json:"systemConfigOwner"`

	GovernanceTokenOwner common.Address `json:"governanceTokenOwner"`

	UnsafeBlockSigner common.Address `json:"unsafeBlockSigner"`

	Batcher common.Address `json:"batcher"`

	Proposer common.Address `json:"proposer"`

	Challenger common.Address `json:"challenger"`
}

func (c *Chain) Check() error {
	if c.ID == 0 {
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
