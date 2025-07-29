package backend

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/config"
	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-faucet/faucet/frontend"
	"github.com/ethereum-optimism/optimism/op-faucet/metrics"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

type Faucet struct {
	mu sync.RWMutex

	log log.Logger
	m   metrics.Metricer

	id       ftypes.FaucetID
	chainID  eth.ChainID
	txMgr    txmgr.TxManager
	elClient apis.EthBalance

	// true when the faucet is disabled and may not serve any new faucet requests
	disabled bool
}

var _ frontend.FaucetBackend = (*Faucet)(nil)

func FaucetFromConfig(logger log.Logger, m metrics.Metricer, fID ftypes.FaucetID, fCfg *config.FaucetEntry) (*Faucet, error) {
	logger = logger.New("faucet", fID, "chain", fCfg.ChainID)
	txCfg, err := fCfg.TxManagerConfig(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to setup tx manager config: %w", err)
	}
	txMgr, err := txmgr.NewSimpleTxManagerFromConfig(string(fID), logger, m, txCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to start tx manager: %w", err)
	}
	elClient, err := ethclient.Dial(fCfg.ELRPC.Value.RPC())
	if err != nil {
		return nil, fmt.Errorf("failed to dial EL client: %w", err)
	}
	return faucetWithTxManager(logger, m, fID, txMgr, elClient), nil
}

func faucetWithTxManager(logger log.Logger, m metrics.Metricer, fID ftypes.FaucetID, txMgr txmgr.TxManager, elClient apis.EthBalance) *Faucet {
	return &Faucet{
		log:      logger,
		m:        m,
		id:       fID,
		chainID:  txMgr.ChainID(),
		txMgr:    txMgr,
		elClient: elClient,
		disabled: false,
	}
}

func (f *Faucet) Enable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log.Info("Enabling faucet")
	f.disabled = false
}

func (f *Faucet) Disable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log.Info("Disabling faucet")
	f.disabled = true
}

func (f *Faucet) Close() {
	f.log.Info("Closing faucet")
	f.Disable()
	f.txMgr.Close()
}

func (f *Faucet) Balance() (eth.ETH, error) {
	wallet := f.txMgr.From()
	balance, err := f.elClient.BalanceAt(context.Background(), wallet, nil)

	var ethBalance eth.ETH
	if err != nil {
		f.log.Error("Failed to get balance", "err", err)
		return ethBalance, err
	}
	return eth.WeiBig(balance), nil
}

func (f *Faucet) ChainID() eth.ChainID {
	return f.chainID
}

func (f *Faucet) RequestETH(ctx context.Context, request *ftypes.FaucetRequest) (result error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	logger := f.log.New("to", request.Target, "amount", request.Amount)
	if f.disabled {
		logger.Info("Cannot serve request, faucet is disabled")
		return errors.New("faucet is disabled")
	}

	logger.Info("Sending funds")

	balance, err := f.Balance()
	if err != nil {
		logger.Warn("Failed to get balance, optimistically continuing the request")
	} else {
		if balance.ToBig().Cmp(request.Amount.ToBig()) < 0 {
			logger.Error("Insufficient balance", "balance", balance.String(), "amount", request.Amount)
			return errors.New("insufficient balance")
		}
	}

	onDone := f.m.RecordFundAction(f.id, f.chainID, request.Amount)
	defer func() {
		onDone(result)
	}()

	// We execute this tiny special EVM program,
	// such that we can move ETH into the target account,
	// without executing the code of the target account.
	// Since we don't want to accidentally trigger untrusted EVM functionality as the funder EOA.

	// This code self-destructs the ephemeral contract-creation-scope,
	// and assigns (no execution!) all the value of this scope to the target address.
	// These types of ephemeral self-destructs are still allowed post-Cancun.
	var out []byte
	out = append(out, byte(vm.PUSH20))
	out = append(out, request.Target[:]...)
	out = append(out, byte(vm.SELFDESTRUCT))

	candidate := txmgr.TxCandidate{
		TxData:   out,
		Blobs:    nil,
		To:       nil, // contract-creation, see above
		GasLimit: 0,   // estimate gas dynamically
		Value:    request.Amount.ToBig(),
	}
	rec, err := f.txMgr.Send(ctx, candidate)
	if err != nil {
		logger.Error("failed to send funds", "err", err)
		return fmt.Errorf("failed to send funds: %w", err)
	}
	if rec.Status == types.ReceiptStatusFailed {
		logger.Error("funding tx reverted", "tx", rec.TxHash)
		return fmt.Errorf("failed to fund, tx %s reverted", rec.TxHash)
	}
	logger.Info("Successfully funded account",
		"tx", rec.TxHash,
		"included_hash", rec.BlockHash,
		"included_num", rec.BlockNumber)
	return nil
}
