package shim

import (
	"crypto/ecdsa"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type UserConfig struct {
	CommonConfig
	ID   stack.UserID
	Priv *ecdsa.PrivateKey
	EL   stack.ELNode
}

type presetUser struct {
	commonImpl
	id   stack.UserID
	priv *ecdsa.PrivateKey
	addr common.Address
	el   stack.ELNode
}

func (p *presetUser) ID() stack.UserID {
	return p.id
}

func (p *presetUser) Address() common.Address {
	return p.addr
}

func (p *presetUser) ChainID() eth.ChainID {
	return p.id.ChainID
}

func (p *presetUser) EL() stack.ELNode {
	return p.el
}

func (p *presetUser) Key() *ecdsa.PrivateKey {
	return p.priv
}

var _ stack.User = (*presetUser)(nil)

func NewUser(cfg UserConfig) stack.User {
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
