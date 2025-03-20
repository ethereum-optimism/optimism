package system2

import "github.com/ethereum/go-ethereum/common"

type SuperchainDeployment interface {
	ProtocolVersionsAddr() common.Address
	SuperchainConfigAddr() common.Address
}

// SuperchainID identifies a Superchain by name, is type-safe, and can be value-copied and used as map key.
type SuperchainID genericID

const SuperchainKind Kind = "Superchain"

func (id SuperchainID) String() string {
	return genericID(id).string(SuperchainKind)
}

func (id SuperchainID) MarshalText() ([]byte, error) {
	return genericID(id).marshalText(SuperchainKind)
}

func (id *SuperchainID) UnmarshalText(data []byte) error {
	return (*genericID)(id).unmarshalText(SuperchainKind, data)
}

func SortSuperchainIDs(ids []SuperchainID) []SuperchainID {
	return copyAndSortCmp(ids)
}

// Superchain is a collection of L2 chains with common rules and shared configuration on L1
type Superchain interface {
	Common
	ID() SuperchainID

	Deployment() SuperchainDeployment
}

type SuperchainConfig struct {
	CommonConfig
	ID         SuperchainID
	Deployment SuperchainDeployment
}

type presetSuperchain struct {
	commonImpl
	id         SuperchainID
	deployment SuperchainDeployment
}

var _ Superchain = (*presetSuperchain)(nil)

func NewSuperchain(cfg SuperchainConfig) Superchain {
	cfg.Log = cfg.Log.New("id", cfg.ID)
	return &presetSuperchain{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		deployment: cfg.Deployment,
	}
}

func (p *presetSuperchain) ID() SuperchainID {
	return p.id
}

func (p presetSuperchain) Deployment() SuperchainDeployment {
	return p.deployment
}
