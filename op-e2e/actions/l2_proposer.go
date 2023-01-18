package actions

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-proposer/proposer"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
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
	signer := func(chainID *big.Int) proposer.SignerFn {
		s := opcrypto.PrivateKeySignerFn(cfg.ProposerKey, chainID)
		return func(_ context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return s(addr, tx)
		}
	}
	from := crypto.PubkeyToAddress(cfg.ProposerKey.PublicKey)
	cliCFG := proposer.CLIConfig{
		L1EthRpc:                  "",
		RollupRpc:                 "",
		L2OOAddress:               "",
		PollInterval:              0,
		NumConfirmations:          0,
		SafeAbortNonceTooLowCount: 0,
		ResubmissionTimeout:       0,
		Mnemonic:                  "",
		L2OutputHDPath:            "",
		PrivateKey:                "",
		RPCConfig:                 rpc.CLIConfig{},
		AllowNonFinalized:         false,
		LogConfig:                 oplog.CLIConfig{},
		MetricsConfig:             metrics.CLIConfig{},
		PprofConfig:               pprof.CLIConfig{},
	}
	dr, err := proposer.NewL2OutputSubmitterWithSigner(cliCFG, from, signer, log)

	require.NoError(t, err)
	return &L2Proposer{
		log:     log,
		l1:      l1,
		driver:  dr,
		address: crypto.PubkeyToAddress(cfg.ProposerKey.PublicKey),
	}
}

func (p *L2Proposer) CanPropose(t Testing) bool {
	_, shouldPropose, err := p.driver.FetchNextOutputInfo(context.TODO())
	require.NoError(t, err)
	return shouldPropose
}

func (p *L2Proposer) ActMakeProposalTx(t Testing) {
	output, shouldPropose, err := p.driver.FetchNextOutputInfo(context.TODO())
	if !shouldPropose {
		return
	}
	require.NoError(t, err)

	tx, err := p.driver.CreateProposalTx(context.TODO(), output)
	require.NoError(t, err)

	err = p.driver.SendTransactionExt(context.TODO(), tx)
	require.NoError(t, err)

	p.lastTx = tx.Hash()
}

func (p *L2Proposer) LastProposalTx() common.Hash {
	return p.lastTx
}
