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
	// L1OperatorKeyDomain is the domain for keys used in L1 ops
	L1OperatorKeyDomain Domain = 1
	// L1UserKeyDomain is the domain for pre-funded user accounts specific to L1
	L1UserKeyDomain Domain = 2
	// SuperchainKeyDomain is the domain for superchain contracts.
	// The L1 chain ID should be used as chain ID for this domain.
	SuperchainKeyDomain Domain = 3
	// L2OperatorKeyDomain is the domain for L2 operator keys
	L2OperatorKeyDomain Domain = 4
	// L2UserKeyDomain is the domain for pre-funded L2 user accounts specific to a single L2
	L2UserKeyDomain Domain = 5
)

func (d Domain) String() string {
	switch d {
	case DefaultKeyDomain:
		return "default"
	case L1OperatorKeyDomain:
		return "l1-operator"
	case L1UserKeyDomain:
		return "l1-user"
	case SuperchainKeyDomain:
		return "superchain"
	case L2OperatorKeyDomain:
		return "l2-operator"
	case L2UserKeyDomain:
		return "l2-user"
	default:
		return fmt.Sprintf("unknown-domain-%d", uint64(d))
	}
}

// Role separates the usage of development keys by exact role, within a Domain.
// Some roles may not be applicable to every Domain, those can be ignored.
type Role uint64

const (
	DeployerRole                   Role = 0
	ProposerRole                   Role = 1
	BatcherRole                    Role = 2
	SequencerP2PRole               Role = 3
	ChallengerRole                 Role = 4
	ProxyAdminOwnerRole            Role = 5
	SuperchainConfigGuardianRole   Role = 6
	FinalSystemOwnerRole           Role = 7
	BaseFeeVaultRecipientRole      Role = 8
	L1FeeVaultRecipientRole        Role = 9
	SequencerFeeVaultRecipientRole Role = 10
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
	case ProxyAdminOwnerRole:
		return "proxy-admin-owner"
	case SuperchainConfigGuardianRole:
		return "superchain-config-guardian"
	case FinalSystemOwnerRole:
		return "final-system-owner"
	case BaseFeeVaultRecipientRole:
		return "base-fee-vault-recipient"
	case L1FeeVaultRecipientRole:
		return "l1-fee-vault-recipient"
	case SequencerFeeVaultRecipientRole:
		return "sequencer-fee-vault-recipient"
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
