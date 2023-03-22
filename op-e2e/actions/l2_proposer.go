package actions

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-proposer/proposer"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

type ProposerCfg struct {
	OutputOracleAddr  common.Address
	ProposerKey       *ecdsa.PrivateKey
	AllowNonFinalized bool
}

type L2Proposer struct {
	log     log.Logger
	l1      *ethclient.Client
	driver  *proposer.L2OutputSubmitter
	address common.Address
	lastTx  common.Hash
}

func NewL2Proposer(t Testing, log log.Logger, cfg *ProposerCfg, l1 *ethclient.Client, rollupCl *sources.RollupClient) *L2Proposer {
	signer := func(chainID *big.Int) opcrypto.SignerFn {
		s := opcrypto.PrivateKeySignerFn(cfg.ProposerKey, chainID)
		return func(_ context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return s(addr, tx)
		}
	}
	from := crypto.PubkeyToAddress(cfg.ProposerKey.PublicKey)

	proposerCfg := proposer.Config{
		L2OutputOracleAddr: cfg.OutputOracleAddr,
		PollInterval:       time.Second,
		TxManagerConfig: txmgr.Config{
			ResubmissionTimeout:       5 * time.Second,
			ReceiptQueryInterval:      time.Second,
			NumConfirmations:          1,
			SafeAbortNonceTooLowCount: 4,
			From:                      from,
			// Signer is loaded in `proposer.NewL2OutputSubmitter`
		},
		L1Client:          l1,
		RollupClient:      rollupCl,
		AllowNonFinalized: cfg.AllowNonFinalized,
		From:              from,
		SignerFnFactory:   signer,
	}

	dr, err := proposer.NewL2OutputSubmitter(proposerCfg, log, metrics.NoopMetrics)
	require.NoError(t, err)

	return &L2Proposer{
		log:     log,
		l1:      l1,
		driver:  dr,
		address: crypto.PubkeyToAddress(cfg.ProposerKey.PublicKey),
	}
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

	tx, err := p.driver.CreateProposalTx(t.Ctx(), output)
	require.NoError(t, err)

	// Note: Use L1 instead of the output submitter's transaction manager because
	// this is non-blocking while the txmgr is blocking & deadlocks the tests
	err = p.l1.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err)

	p.lastTx = tx.Hash()
}

func (p *L2Proposer) LastProposalTx() common.Hash {
	return p.lastTx
}
