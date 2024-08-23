package devkeys

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Domain separates the usage of development keys by broad kind of usage.
// E.g. a general user account, a L1 thing, a superchain thing, a L2 thing, etc.
type Domain uint64

const (
	// DefaultKeyDomain is the domain for keys that get pre-funded in every chain.
	// A 0 chain ID should be used for this domain.
	DefaultKeyDomain Domain = 0
	// OperatorKeyDomain is the domain for keys used in operation of a chain.
	// The key may be used on any chain, the chain ID represents the chain that operations are for.
	OperatorKeyDomain Domain = 1
	// UserKeyDomain is the domain for pre-funded user accounts specific to a single chain, not initially funded in other chains.
	UserKeyDomain Domain = 2
	// SuperchainKeyDomain is the domain for superchain contracts.
	// The chain ID that the SuperchainConfig is deployed on should be used as chain ID for this domain.
	SuperchainKeyDomain Domain = 3
)

func (d Domain) String() string {
	switch d {
	case DefaultKeyDomain:
		return "default"
	case OperatorKeyDomain:
		return "operator"
	case UserKeyDomain:
		return "user"
	case SuperchainKeyDomain:
		return "superchain"
	default:
		return fmt.Sprintf("unknown-domain-%d", uint64(d))
	}
}

// Role separates the usage of development keys by exact role, within a Domain.
// Some roles may not be applicable to every Domain, those can be ignored.
type Role uint64

const (
	// DeployerRole is the deployer of contracts
	DeployerRole Role = 0
	// ProposerRole is the key used by op-proposer
	ProposerRole Role = 1
	// BatcherRole is the key used by op-batcher
	BatcherRole Role = 2
	// SequencerP2PRole is the key used to publish sequenced L2 blocks
	SequencerP2PRole Role = 3
	// ChallengerRole is the key used by op-challenger
	ChallengerRole Role = 4
	// L2ProxyAdminOwnerRole is the key that controls the ProxyAdmin predeploy in L2
	L2ProxyAdminOwnerRole Role = 5
	// SuperchainConfigGuardianRole is the Guardian of the SuperchainConfig
	SuperchainConfigGuardianRole Role = 6
	// L1ProxyAdminOwnerRole is the key that owns the ProxyAdmin on the L1 side of the deployment.
	// This can be the ProxyAdmin of a L2 chain deployment, or a superchain deployment, depending on the domain.
	L1ProxyAdminOwnerRole Role = 7
	// BaseFeeVaultRecipientRole is the key that receives from the BaseFeeVault predeploy
	BaseFeeVaultRecipientRole Role = 8
	// L1FeeVaultRecipientRole is the key that receives from the L1FeeVault predeploy
	L1FeeVaultRecipientRole Role = 9
	// SequencerFeeVaultRecipientRole is the key that receives form the SequencerFeeVault predeploy
	SequencerFeeVaultRecipientRole Role = 10
	// DependencySetManagerRole is the key used to manage the dependency set of a superchain.
	DependencySetManagerRole Role = 11
)

func (d Role) String() string {
	switch d {
	case DeployerRole:
		return "deployer"
	case ProposerRole:
		return "proposer"
	case BatcherRole:
		return "batcher"
	case SequencerP2PRole:
		return "sequencer-p2p"
	case ChallengerRole:
		return "challenger"
	case L2ProxyAdminOwnerRole:
		return "l2-proxy-admin-owner"
	case SuperchainConfigGuardianRole:
		return "superchain-config-guardian"
	case L1ProxyAdminOwnerRole:
		return "l1-proxy-admin-owner"
	case BaseFeeVaultRecipientRole:
		return "base-fee-vault-recipient"
	case L1FeeVaultRecipientRole:
		return "l1-fee-vault-recipient"
	case SequencerFeeVaultRecipientRole:
		return "sequencer-fee-vault-recipient"
	case DependencySetManagerRole:
		return "dependency-set-manager"
	default:
		return fmt.Sprintf("unknown-role-%d", uint64(d))
	}
}

// DevSecrets selects a secret-key based on domain, chain and role.
// This is meant for dev-purposes only.
// Secret keys should not directly be exposed to live production services.
type DevSecrets interface {
	Secret(domain Domain, chain *big.Int, role Role) (*ecdsa.PrivateKey, error)
}

// DevAddresses selects an address based on domain, chain and role.
// This interface is preferred in tools that do not directly rely on secret-key material.
type DevAddresses interface {
	// Address produces an address for the given domain/role.
	Address(domain Domain, chain *big.Int, role Role) (common.Address, error)
}

// DevKeys is a joint interface of DevSecrets and DevAddresses
type DevKeys interface {
	DevSecrets
	DevAddresses
}
