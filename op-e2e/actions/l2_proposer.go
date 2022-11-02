package actions

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-proposer/drivers/l2output"
)

type ProposerCfg struct {
	OutputOracleAddr  common.Address
	ProposerKey       *ecdsa.PrivateKey
	AllowNonFinalized bool
}

type L2Proposer struct {
	log     log.Logger
	l1      *ethclient.Client
	driver  *l2output.Driver
	address common.Address
	lastTx  common.Hash
}

func NewL2Proposer(t Testing, log log.Logger, cfg *ProposerCfg, l1 *ethclient.Client, rollupCl *sources.RollupClient) *L2Proposer {
	chainID, err := l1.ChainID(t.Ctx())
	require.NoError(t, err)
	dr, err := l2output.NewDriver(l2output.Config{
		Log:               log,
		Name:              "proposer",
		L1Client:          l1,
		RollupClient:      rollupCl,
		AllowNonFinalized: cfg.AllowNonFinalized,
		L2OOAddr:          cfg.OutputOracleAddr,
		ChainID:           chainID,
		PrivKey:           cfg.ProposerKey,
	})
	require.NoError(t, err)
	return &L2Proposer{
		log:     log,
		l1:      l1,
		driver:  dr,
		address: crypto.PubkeyToAddress(cfg.ProposerKey.PublicKey),
	}
}

func (p *L2Proposer) CanPropose(t Testing) bool {
	start, end, err := p.driver.GetBlockRange(t.Ctx())
	require.NoError(t, err)
	return start.Cmp(end) < 0
}

func (p *L2Proposer) ActMakeProposalTx(t Testing) {
	start, end, err := p.driver.GetBlockRange(t.Ctx())
	require.NoError(t, err)
	if start.Cmp(end) == 0 {
		t.InvalidAction("nothing to propose, block range starts and ends at %s", start.String())
	}
	nonce, err := p.l1.PendingNonceAt(t.Ctx(), p.address)
	require.NoError(t, err)

	tx, err := p.driver.CraftTx(t.Ctx(), start, end, new(big.Int).SetUint64(nonce))
	require.NoError(t, err)

	err = p.driver.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err)
	p.lastTx = tx.Hash()
}

func (p *L2Proposer) LastProposalTx() common.Hash {
	return p.lastTx
}
