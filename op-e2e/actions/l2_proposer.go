package actions

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

type ProposerCfg struct {
	OutputOracleAddr  common.Address
	ProposerKey       *ecdsa.PrivateKey
	AllowNonFinalized bool
}

type L2Proposer struct {
	log          log.Logger
	l1           *ethclient.Client
	driver       *proposer.L2OutputSubmitter
	address      common.Address
	privKey      *ecdsa.PrivateKey
	contractAddr common.Address
	lastTx       common.Hash
}

type fakeTxMgr struct {
	from common.Address
}

func (f fakeTxMgr) From() common.Address {
	return f.from
}
func (f fakeTxMgr) Send(_ context.Context, _ txmgr.TxCandidate) (*types.Receipt, error) {
	panic("unimplemented")
}

func NewL2Proposer(t Testing, log log.Logger, cfg *ProposerCfg, l1 *ethclient.Client, rollupCl *sources.RollupClient) *L2Proposer {

	proposerCfg := proposer.Config{
		L2OutputOracleAddr: cfg.OutputOracleAddr,
		PollInterval:       time.Second,
		L1Client:           l1,
		RollupClient:       rollupCl,
		AllowNonFinalized:  cfg.AllowNonFinalized,
		// We use custom signing here instead of using the transaction manager.
		TxManager: fakeTxMgr{from: crypto.PubkeyToAddress(cfg.ProposerKey.PublicKey)},
	}

	dr, err := proposer.NewL2OutputSubmitter(proposerCfg, log, metrics.NoopMetrics)
	require.NoError(t, err)

	return &L2Proposer{
		log:          log,
		l1:           l1,
		driver:       dr,
		address:      crypto.PubkeyToAddress(cfg.ProposerKey.PublicKey),
		privKey:      cfg.ProposerKey,
		contractAddr: cfg.OutputOracleAddr,
	}
}

// sendTx reimplements creating & sending transactions because we need to do the final send as async in
// the action tests while we do it synchronously in the real system.
func (p *L2Proposer) sendTx(t Testing, data []byte) {
	gasTipCap := big.NewInt(2 * params.GWei)
	pendingHeader, err := p.l1.HeaderByNumber(t.Ctx(), big.NewInt(-1))
	require.NoError(t, err, "need l1 pending header for gas price estimation")
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(pendingHeader.BaseFee, big.NewInt(2)))
	chainID, err := p.l1.ChainID(t.Ctx())
	require.NoError(t, err)
	nonce, err := p.l1.NonceAt(t.Ctx(), p.address, nil)
	require.NoError(t, err)

	gasLimit, err := p.l1.EstimateGas(t.Ctx(), ethereum.CallMsg{
		From:      p.address,
		To:        &p.contractAddr,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Data:      data,
	})
	require.NoError(t, err)

	rawTx := &types.DynamicFeeTx{
		Nonce:     nonce,
		To:        &p.contractAddr,
		Data:      data,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Gas:       gasLimit,
		ChainID:   chainID,
	}

	tx, err := types.SignNewTx(p.privKey, types.LatestSignerForChainID(chainID), rawTx)
	require.NoError(t, err, "need to sign tx")

	err = p.l1.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "need to send tx")

	p.lastTx = tx.Hash()
}

func (p *L2Proposer) CanPropose(t Testing) bool {
	_, shouldPropose, err := p.driver.FetchNextOutputInfo(t.Ctx())
	require.NoError(t, err)
	return shouldPropose
}

func (p *L2Proposer) ActMakeProposalTx(t Testing) {
	output, shouldPropose, err := p.driver.FetchNextOutputInfo(t.Ctx())
	if !shouldPropose {
		return
	}
	require.NoError(t, err)

	txData, err := p.driver.ProposeL2OutputTxData(output)
	require.NoError(t, err)

	// Note: Use L1 instead of the output submitter's transaction manager because
	// this is non-blocking while the txmgr is blocking & deadlocks the tests
	p.sendTx(t, txData)
}

func (p *L2Proposer) LastProposalTx() common.Hash {
	return p.lastTx
}
