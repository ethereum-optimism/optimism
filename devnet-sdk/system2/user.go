package system2

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// UserID identifies a User by name and chainID, is type-safe, and can be value-copied and used as map key.
type UserID idWithChain

const UserKind Kind = "User"

func (id UserID) String() string {
	return idWithChain(id).string(UserKind)
}

func (id UserID) MarshalText() ([]byte, error) {
	return idWithChain(id).marshalText(UserKind)
}

func (id *UserID) UnmarshalText(data []byte) error {
	return (*idWithChain)(id).unmarshalText(UserKind, data)
}

func SortUserIDs(ids []UserID) []UserID {
	return copyAndSort(ids, func(a, b UserID) bool {
		return lessIDWithChain(idWithChain(a), idWithChain(b))
	})
}

// User represents a single user-key, specific to a single chain,
// with a default connection to interact with the execution-layer of said chain.
type User interface {
	Common

	ID() UserID

	Key() *ecdsa.PrivateKey
	Address() common.Address

	ChainID() eth.ChainID

	// EL is the default node used to interact with the chain
	EL() ELNode
}

type UserConfig struct {
	CommonConfig
	ID   UserID
	Priv *ecdsa.PrivateKey
	EL   ELNode
}

type presetUser struct {
	commonImpl
	id   UserID
	priv *ecdsa.PrivateKey
	addr common.Address
	el   ELNode
}

func (p *presetUser) ID() UserID {
	return p.id
}

func (p *presetUser) Address() common.Address {
	return p.addr
}

func (p *presetUser) ChainID() eth.ChainID {
	return p.id.ChainID
}

func (p *presetUser) EL() ELNode {
	return p.el
}

func (p *presetUser) Key() *ecdsa.PrivateKey {
	return p.priv
}

var _ User = (*presetUser)(nil)

func NewUser(cfg UserConfig) User {
	require.Equal(cfg.T, cfg.ID.ChainID, cfg.EL.ChainID(), "user must be on the same chain as the EL node")
	cfg.Log = cfg.Log.New("chainID", cfg.ID.ChainID, "id", cfg.ID)
	return &presetUser{
		commonImpl: newCommon(cfg.CommonConfig),
		id:         cfg.ID,
		priv:       cfg.Priv,
		addr:       crypto.PubkeyToAddress(cfg.Priv.PublicKey),
		el:         cfg.EL,
	}
}
