package driver

import (
	"context"
	"errors"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/submit"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"sync"
	"time"
)

type DAHash struct {
	hash  common.Hash
	count uint64
}
type DAManager struct {
	log               log.Logger
	engine            derive.ResettableEngineControl
	wg                sync.WaitGroup
	shutdownCtx       context.Context
	cancelShutdownCtx context.CancelFunc
	hashCh            chan common.Hash
	daHashes          map[common.Hash]uint8

	TxMgr        *txmgr.SimpleTxManager
	RollupConfig *rollup.Config

	IsBroadcast bool
}

func NewDAManager(log log.Logger, rollup *rollup.Config, engine derive.ResettableEngineControl, txmgr *txmgr.SimpleTxManager, isBroadcast bool) *DAManager {
	return &DAManager{
		log:          log,
		engine:       engine,
		TxMgr:        txmgr,
		RollupConfig: rollup,
		IsBroadcast:  isBroadcast,
		hashCh:       make(chan common.Hash),
		daHashes:     make(map[common.Hash]uint8),
	}
}

func (d *DAManager) SendDA(ctx context.Context, index, length uint64, broadcaster, user common.Address, commitment, sign, data []byte) (common.Hash, error) {
	if !d.IsBroadcast {
		return common.Hash{}, errors.New("broadcast node not started")
	}
	//d.log.Info("SendDA", "index", index, "length", length, "broadcaster", broadcaster.Hex(), "user", user.Hex(), "commitment", commitment, "sign", sign, "data", data)
	if !verifySignature(index, length, broadcaster, user, commitment, sign) {
		return common.Hash{}, errors.New("invalid public key")
	}
	input, err := submit.L1SubmitTxData(user, uint64(index), commitment, sign)
	if err != nil {
		return common.Hash{}, err
	}
	log.Info("L1SubmitTxData")

	tx, err := d.TxMgr.SendDA(ctx, txmgr.TxCandidate{
		TxData:   input,
		To:       &d.RollupConfig.SubmitContractAddress,
		GasLimit: 0,
	})
	log.Info("Send")

	if err != nil {
		return common.Hash{}, err
	}
	log.Info("L1Submit tx successfully published",
		"tx_hash", tx.Hash().Hex())

	d.engine.UploadFileDataByParams(ctx, index, length, broadcaster, user, commitment, sign, data, tx.Hash())
	return tx.Hash(), nil
}

func (d *DAManager) Broadcaster(ctx context.Context) (common.Address, error) {
	if d.IsBroadcast {
		return d.TxMgr.From(), nil
	}
	return common.Address{}, errors.New("broadcast node not started")
}

func (d *DAManager) Start() bool {
	d.shutdownCtx, d.cancelShutdownCtx = context.WithCancel(context.Background())
	d.wg.Add(1)
	go d.loop()
	return true
}

func (d *DAManager) Test() bool {
	d.hashCh <- common.HexToHash("0x059e4161e765a2af3eed83187aa3da8a35f839617b4847a9cb46f71e8cccd670")
	return true
}

func (d *DAManager) loop() {
	defer d.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	//receiptsCh := make(chan DAHash)
	//queue := txmgr.NewQueue[txData](l.killCtx, l.Txmgr, l.Config.MaxPendingTransactions)

	for {
		select {
		case <-ticker.C:
			d.getDA()
		case test := <-d.hashCh:
			d.daHashes[test] = 0
		case <-d.shutdownCtx.Done():
			d.log.Info("d.shutdownCtx.Done()")
			return
		}
	}
}

func (d *DAManager) getDA() {
	if len(d.daHashes) < 1 {
		return
	}
	for hash, count := range d.daHashes {
		if count == 4 {
			delete(d.daHashes, hash)
			continue
		}
		log.Info("getDA", "hash", hash, "count", count)
		d.daHashes[hash] = count + 1
	}
}
func verifySignature(index, length uint64, broadcaster, user common.Address, commitment, sign []byte) bool {
	return true
}
