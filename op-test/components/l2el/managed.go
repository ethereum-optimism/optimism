package l2el

import (
	"crypto/rand"
	"os"
	"path"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethEth "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-test/components/l2"
	"github.com/ethereum-optimism/optimism/op-test/test"
)

type ManagedOpGeth struct {
	l2 l2.L2

	backend *gethEth.Ethereum
	node    *node.Node

	jwtSecret [32]byte
}

func (m *ManagedOpGeth) HTTPAuthEndpoint() string {
	return m.node.HTTPAuthEndpoint()
}

func (m *ManagedOpGeth) WSAuthEndpoint() string {
	return m.node.WSAuthEndpoint()
}

func (m *ManagedOpGeth) JWTSecret() [32]byte {
	return m.jwtSecret
}

func (m *ManagedOpGeth) WSEndpoint() string {
	return m.node.WSEndpoint()
}

func (m *ManagedOpGeth) HTTPEndpoint() string {
	return m.node.HTTPEndpoint()
}

func (m *ManagedOpGeth) RPC() *rpc.Client {
	return m.node.Attach()
}

func (m *ManagedOpGeth) L2() l2.L2 {
	return m.l2
}

var _ L2EL = (*ManagedOpGeth)(nil)

func NewManagedOpGeth(t test.Testing, l2Chain l2.L2) *ManagedOpGeth {
	var jwtSecret [32]byte
	_, err := rand.Read(jwtSecret[:])
	require.NoError(t, err)

	// Sadly the geth node config cannot load JWT secret from memory, it has to be a file
	jwtPath := path.Join(t.TempDir(), "jwt_secret")
	require.NoError(t, os.WriteFile(jwtPath, []byte(hexutil.Encode(jwtSecret[:])), 0600), "must write jwt secret")

	chainID := l2Chain.ChainID()
	genesis := l2Chain.Genesis()
	var gethOpts []geth.GethOption // TODO
	gethNode, gethBackend, err := geth.InitL2("managed-op-geth", chainID, genesis, jwtPath, gethOpts...)
	require.NoError(t, err, "failed to setup op-geth")

	return &ManagedOpGeth{
		l2:        l2Chain,
		backend:   gethBackend,
		node:      gethNode,
		jwtSecret: [32]byte{},
	}
}
