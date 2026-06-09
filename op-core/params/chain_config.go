// Package params holds the OP-Stack-specific chain configuration types. Consumers
// import it as opparams.
package params

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/forks"
)

const (
	// OPMainnetChainID is the chain ID of OP Mainnet, which has pre-Regolith
	// upgrade history that needs special handling when building its config.
	OPMainnetChainID = 10
	// OPMainnetGenesisBlockNum is the Bedrock-migration block of OP Mainnet, which
	// it treats as its genesis block.
	OPMainnetGenesisBlockNum = 105235063
)

// OptimismConfig holds the OP-Stack-specific EIP-1559 parameters of a chain.
//
// The JSON field names mirror op-geth's params.OptimismConfig verbatim: the value
// is carried as rollup.Config.ChainOpConfig, which is serialised on the wire, so the
// encoding must stay byte-for-byte identical.
//
// Unlike other op-core types, OptimismConfig keeps its Optimism prefix: it is the
// type of the ChainConfig.Optimism field, and the type name pairs with the field.
type OptimismConfig struct {
	EIP1559Elasticity        uint64  `json:"eip1559Elasticity"`
	EIP1559Denominator       uint64  `json:"eip1559Denominator"`
	EIP1559DenominatorCanyon *uint64 `json:"eip1559DenominatorCanyon,omitempty"`
}

// String implements fmt.Stringer.
func (o *OptimismConfig) String() string {
	return "optimism"
}

// ChainConfig is the OP-Stack hardfork schedule and configuration of a chain. It is
// a standalone type: it does not embed go-ethereum's params.ChainConfig, so it carries
// only OP-specific state and the predicates below. Code that needs a go-ethereum config
// — to drive an EVM, genesis, or block processor — converts via GethChainConfig.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"`

	Optimism *OptimismConfig `json:"optimism,omitempty"`

	BedrockBlock *big.Int `json:"bedrockBlock,omitempty"` // Bedrock switch block (nil = no fork, 0 = already on optimism bedrock)
	RegolithTime *uint64  `json:"regolithTime,omitempty"` // Regolith switch time (nil = no fork, 0 = already on optimism regolith)
	CanyonTime   *uint64  `json:"canyonTime,omitempty"`   // Canyon switch time (nil = no fork, 0 = already on optimism canyon)
	EcotoneTime  *uint64  `json:"ecotoneTime,omitempty"`  // Ecotone switch time (nil = no fork, 0 = already on optimism ecotone)
	FjordTime    *uint64  `json:"fjordTime,omitempty"`    // Fjord switch time (nil = no fork, 0 = already on optimism fjord)
	GraniteTime  *uint64  `json:"graniteTime,omitempty"`  // Granite switch time (nil = no fork, 0 = already on optimism granite)
	HoloceneTime *uint64  `json:"holoceneTime,omitempty"` // Holocene switch time (nil = no fork, 0 = already on optimism holocene)
	IsthmusTime  *uint64  `json:"isthmusTime,omitempty"`  // Isthmus switch time (nil = no fork, 0 = already on optimism isthmus)
	JovianTime   *uint64  `json:"jovianTime,omitempty"`   // Jovian switch time (nil = no fork, 0 = already on optimism jovian)
	KarstTime    *uint64  `json:"karstTime,omitempty"`    // Karst switch time (nil = no fork, 0 = already on optimism karst)
	InteropTime  *uint64  `json:"interopTime,omitempty"`  // Interop switch time (nil = no fork, 0 = already on optimism interop)
}

// ActivationTime returns the activation timestamp of the given timestamp-based
// OP-Stack fork, or nil if the fork is not scheduled on this chain. It is the single
// source of truth that the Is<Fork> predicates below delegate to; to add a fork,
// extend this switch and its SetActivationTime counterpart.
//
// Only forks with a timestamp field on the chain config are covered. Bedrock is
// block-based — use IsBedrock — and forks without an execution-layer representation
// here (e.g. Delta) are not handled and cause a panic, mirroring op-geth's config,
// which has no field for them.
func (c *ChainConfig) ActivationTime(fork forks.Name) *uint64 {
	switch fork {
	case forks.Lagoon: // the Interop fork is named Lagoon in the forks package
		return c.InteropTime
	case forks.Karst:
		return c.KarstTime
	case forks.Jovian:
		return c.JovianTime
	case forks.Isthmus:
		return c.IsthmusTime
	case forks.Holocene:
		return c.HoloceneTime
	case forks.Granite:
		return c.GraniteTime
	case forks.Fjord:
		return c.FjordTime
	case forks.Ecotone:
		return c.EcotoneTime
	case forks.Canyon:
		return c.CanyonTime
	case forks.Regolith:
		return c.RegolithTime
	default:
		panic(fmt.Sprintf("ActivationTime: unsupported fork: %v", fork))
	}
}

// SetActivationTime sets the activation timestamp of the given timestamp-based
// OP-Stack fork. It is the setter counterpart to ActivationTime; the two switches
// are the only places that map a fork to its field.
func (c *ChainConfig) SetActivationTime(fork forks.Name, timestamp *uint64) {
	switch fork {
	case forks.Lagoon:
		c.InteropTime = timestamp
	case forks.Karst:
		c.KarstTime = timestamp
	case forks.Jovian:
		c.JovianTime = timestamp
	case forks.Isthmus:
		c.IsthmusTime = timestamp
	case forks.Holocene:
		c.HoloceneTime = timestamp
	case forks.Granite:
		c.GraniteTime = timestamp
	case forks.Fjord:
		c.FjordTime = timestamp
	case forks.Ecotone:
		c.EcotoneTime = timestamp
	case forks.Canyon:
		c.CanyonTime = timestamp
	case forks.Regolith:
		c.RegolithTime = timestamp
	default:
		panic(fmt.Sprintf("SetActivationTime: unsupported fork: %v", fork))
	}
}

// IsForkActive returns whether the given timestamp-based fork is active at or after
// the given timestamp.
func (c *ChainConfig) IsForkActive(fork forks.Name, timestamp uint64) bool {
	activationTime := c.ActivationTime(fork)
	return activationTime != nil && timestamp >= *activationTime
}

// IsRegolith returns whether the Regolith fork is active at the given timestamp.
func (c *ChainConfig) IsRegolith(time uint64) bool { return c.IsForkActive(forks.Regolith, time) }

// IsCanyon returns whether the Canyon fork is active at the given timestamp.
func (c *ChainConfig) IsCanyon(time uint64) bool { return c.IsForkActive(forks.Canyon, time) }

// IsEcotone returns whether the Ecotone fork is active at the given timestamp.
func (c *ChainConfig) IsEcotone(time uint64) bool { return c.IsForkActive(forks.Ecotone, time) }

// IsFjord returns whether the Fjord fork is active at the given timestamp.
func (c *ChainConfig) IsFjord(time uint64) bool { return c.IsForkActive(forks.Fjord, time) }

// IsGranite returns whether the Granite fork is active at the given timestamp.
func (c *ChainConfig) IsGranite(time uint64) bool { return c.IsForkActive(forks.Granite, time) }

// IsHolocene returns whether the Holocene fork is active at the given timestamp.
func (c *ChainConfig) IsHolocene(time uint64) bool { return c.IsForkActive(forks.Holocene, time) }

// IsIsthmus returns whether the Isthmus fork is active at the given timestamp.
func (c *ChainConfig) IsIsthmus(time uint64) bool { return c.IsForkActive(forks.Isthmus, time) }

// IsJovian returns whether the Jovian fork is active at the given timestamp.
func (c *ChainConfig) IsJovian(time uint64) bool { return c.IsForkActive(forks.Jovian, time) }

// IsKarst returns whether the Karst fork is active at the given timestamp.
func (c *ChainConfig) IsKarst(time uint64) bool { return c.IsForkActive(forks.Karst, time) }

// IsInterop returns whether the Interop fork is active at the given timestamp.
func (c *ChainConfig) IsInterop(time uint64) bool { return c.IsForkActive(forks.Lagoon, time) }

// IsBedrock returns whether num is at or after the Bedrock fork block.
func (c *ChainConfig) IsBedrock(num *big.Int) bool {
	return isBlockForked(c.BedrockBlock, num)
}

// isBlockForked returns whether a fork scheduled at block s is active at the given
// head block.
func isBlockForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}
