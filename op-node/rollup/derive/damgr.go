package derive

import (
	"context"
	"errors"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/submit"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"sync"
	"time"
)

type DABlockInfo struct {
	rpc.TxHashes
	Height uint64
	Hash   common.Hash
}
type DAManager struct {
	log               log.Logger
	engine            Engine
	wg                sync.WaitGroup
	shutdownCtx       context.Context
	cancelShutdownCtx context.CancelFunc
	hashCh            chan *DABlockInfo
	daHashes          map[*DABlockInfo]uint8

	TxMgr        *txmgr.SimpleTxManager
	RollupConfig *rollup.Config

	IsBroadcast bool
}

func NewDAManager(log log.Logger, rollup *rollup.Config, engine Engine, txmgr *txmgr.SimpleTxManager, isBroadcast bool) *DAManager {
	return &DAManager{
		log:          log,
		engine:       engine,
		TxMgr:        txmgr,
		RollupConfig: rollup,
		IsBroadcast:  isBroadcast,
		hashCh:       make(chan *DABlockInfo),
		daHashes:     make(map[*DABlockInfo]uint8),
	}
}

func (d *DAManager) SendDA(ctx context.Context, index, length uint64, broadcaster, user common.Address, commitment, sign, data []byte) (common.Hash, error) {
	if !d.IsBroadcast {
		return common.Hash{}, errors.New("broadcast node not started")
	}
	if !verifySignature(index, length, broadcaster, user, commitment, sign) {
		return common.Hash{}, errors.New("invalid public key")
	}
	input, err := submit.L1SubmitTxData(index, length, user, sign, commitment)
	if err != nil {
		log.Info("L1SubmitTxData", "err", err)
		return common.Hash{}, err
	}
	log.Info("L1SubmitTxData")

	tx, err := d.TxMgr.SendDA(ctx, txmgr.TxCandidate{
		TxData:   input,
		To:       &d.RollupConfig.SubmitContractAddress,
		GasLimit: 0,
	})

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

func (d *DAManager) SendDaHash(hash *DABlockInfo) bool {
	d.hashCh <- hash
	return true
}

func (d *DAManager) ChangeCurrentState(stats uint64, number uint64) {
	d.engine.ChangeCurrentState(d.shutdownCtx, stats, rpc.BlockNumber(number))
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
		case hash := <-d.hashCh:
			d.daHashes[hash] = 0
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
	for block, count := range d.daHashes {

		data, err := d.engine.BatchFileDataByHashes(d.shutdownCtx, block.TxHashes)
		if err != nil {
			log.Error("getDA", "height", block.Height, "err", err)
		} else {
			filteredHashes := make([]common.Hash, 0)
			exHashes := make([]common.Hash, 0)
			for i, exists := range data.Flags {
				hash := block.TxHashes.TxHashes[i]
				if exists {
					exHashes = append(exHashes, hash)
				} else {
					filteredHashes = append(filteredHashes, hash)
				}
			}
			block.TxHashes.TxHashes = filteredHashes
			if len(exHashes) > 0 {
				d.engine.BatchSaveFileDataWithHashes(d.shutdownCtx, rpc.TxHashes{TxHashes: exHashes, BlockHash: block.Hash, BlockNumber: rpc.BlockNumber(block.Height)})
			}
		}
		d.daHashes[block] = count + 1
		if count == 5 {
			log.Info("block processed timeout", "helght", block.Height)
			d.engine.ChangeCurrentState(d.shutdownCtx, 3, rpc.BlockNumber(block.Height))
			delete(d.daHashes, block)
		}
		if len(block.TxHashes.TxHashes) == 0 {
			log.Info("block processed", "helght", block.Height)
			d.engine.ChangeCurrentState(d.shutdownCtx, 2, rpc.BlockNumber(block.Height))
			delete(d.daHashes, block)
		}
	}
}
func verifySignature(index, length uint64, broadcaster, user common.Address, commitment, sign []byte) bool {
	return true
}
