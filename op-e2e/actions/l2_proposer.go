package actions

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/sources"
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
	contract     *bindings.L2OutputOracleCaller
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
func (f fakeTxMgr) BlockNumber(_ context.Context) (uint64, error) {
	panic("unimplemented")
}
func (f fakeTxMgr) Send(_ context.Context, _ txmgr.TxCandidate) (*types.Receipt, error) {
	panic("unimplemented")
}
func (f fakeTxMgr) Close() {
}

func NewL2Proposer(t Testing, log log.Logger, cfg *ProposerCfg, l1 *ethclient.Client, rollupCl *sources.RollupClient) *L2Proposer {
	proposerConfig := proposer.ProposerConfig{
		PollInterval:       time.Second,
		NetworkTimeout:     time.Second,
		L2OutputOracleAddr: cfg.OutputOracleAddr,
		AllowNonFinalized:  cfg.AllowNonFinalized,
	}
	rollupProvider, err := dial.NewStaticL2RollupProviderFromExistingRollup(rollupCl)
	require.NoError(t, err)
	driverSetup := proposer.DriverSetup{
		Log:            log,
		Metr:           metrics.NoopMetrics,
		Cfg:            proposerConfig,
		Txmgr:          fakeTxMgr{from: crypto.PubkeyToAddress(cfg.ProposerKey.PublicKey)},
		L1Client:       l1,
		RollupProvider: rollupProvider,
	}

	dr, err := proposer.NewL2OutputSubmitter(driverSetup)
	require.NoError(t, err)
	contract, err := bindings.NewL2OutputOracleCaller(cfg.OutputOracleAddr, l1)
	require.NoError(t, err)

	address := crypto.PubkeyToAddress(cfg.ProposerKey.PublicKey)
	proposer, err := contract.PROPOSER(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, proposer, address, "PROPOSER must be the proposer's address")

	return &L2Proposer{
		log:          log,
		l1:           l1,
		driver:       dr,
		contract:     contract,
		address:      address,
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

	gasLimit, err := estimateGasPending(t.Ctx(), p.l1, ethereum.CallMsg{
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
	log.Info("Proposer sent tx", "hash", tx.Hash(), "to", p.contractAddr)
	require.NoError(t, err, "need to send tx")

	p.lastTx = tx.Hash()
}

// estimateGasPending calls eth_estimateGas specifying the pending block. This is required for transactions from the
// proposer because they include a reference to the latest block which isn't available via `BLOCKHASH` if `latest` is
// used. In production code, the proposer waits until another L1 block is published but in e2e tests no new L1 blocks
// will be created so pending must be used.
func estimateGasPending(ctx context.Context, ec *ethclient.Client, msg ethereum.CallMsg) (uint64, error) {
	var hex hexutil.Uint64
	err := ec.Client().CallContext(ctx, &hex, "eth_estimateGas", toCallArg(msg), "pending")
	if err != nil {
		return 0, err
	}
	return uint64(hex), nil
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
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
