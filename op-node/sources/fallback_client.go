package sources

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

// FallbackClient is an RPC client, it can automatically switch to the next endpoint
// when there is a problem with the current endpoint
// and automatically switch back after the first endpoint recovers.
type FallbackClient struct {
	// firstRpc is created by the first of the urls, it should be used first in a healthy state
	firstRpc          client.RPC
	urlList           []string
	rpcInitFunc       func(url string) (client.RPC, error)
	lastMinuteFail    atomic.Int64
	currentRpc        atomic.Pointer[client.RPC]
	currentIndex      int
	mx                sync.Mutex
	log               log.Logger
	isInFallbackState bool
	subscribeFunc     func() (event.Subscription, error)
	headsSub          *ethereum.Subscription
	chainId           *big.Int
	genesisBlock      eth.BlockID
	ctx               context.Context
	isClose           chan struct{}
	metrics           metrics.Metricer
}

const threshold int64 = 20

// NewFallbackClient returns a new FallbackClient. chainId and genesisBlock are used to check
// whether the newly switched rpc is legal.
func NewFallbackClient(ctx context.Context, rpc client.RPC, urlList []string, log log.Logger, chainId *big.Int, genesisBlock eth.BlockID, rpcInitFunc func(url string) (client.RPC, error)) client.RPC {
	fallbackClient := &FallbackClient{
		ctx:          ctx,
		firstRpc:     rpc,
		urlList:      urlList,
		log:          log,
		rpcInitFunc:  rpcInitFunc,
		currentIndex: 0,
		chainId:      chainId,
		genesisBlock: genesisBlock,
	}
	fallbackClient.currentRpc.Store(&rpc)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for {
			select {
			case <-ticker.C:
				log.Debug("FallbackClient clear lastMinuteFail 0")
				fallbackClient.lastMinuteFail.Store(0)
			case <-fallbackClient.isClose:
				return
			default:
				if fallbackClient.lastMinuteFail.Load() >= threshold {
					fallbackClient.switchCurrentRpc()
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
	return fallbackClient
}

func (l *FallbackClient) Close() {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.isClose <- struct{}{}
	currentRpc := *l.currentRpc.Load()
	currentRpc.Close()
	if currentRpc != l.firstRpc {
		l.firstRpc.Close()
	}
}

func (l *FallbackClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	err := (*l.currentRpc.Load()).CallContext(ctx, result, method, args...)
	if err != nil {
		l.handleErr(err)
	}
	return err
}

func (l *FallbackClient) handleErr(err error) {
	if errors.Is(err, rpc.ErrNoResult) {
		return
	}
	var targetErr rpc.Error
	if errors.As(err, &targetErr) {
		return
	}
	l.lastMinuteFail.Add(1)
}

func (l *FallbackClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	err := (*l.currentRpc.Load()).BatchCallContext(ctx, b)
	if err != nil {
		l.handleErr(err)
	}
	return err
}

func (l *FallbackClient) EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error) {
	subscribe, err := (*l.currentRpc.Load()).EthSubscribe(ctx, channel, args...)
	if err != nil {
		l.handleErr(err)
	}
	return subscribe, err
}

func (l *FallbackClient) switchCurrentRpc() {
	if l.currentIndex >= len(l.urlList) {
		l.log.Error("the fallback client has tried all urls, but all failed")
		return
	}
	l.mx.Lock()
	defer l.mx.Unlock()
	if l.lastMinuteFail.Load() <= threshold {
		return
	}
	if l.metrics != nil {
		l.metrics.RecordUrlSwitchEvent()
	}
	for {
		l.currentIndex++
		if l.currentIndex >= len(l.urlList) {
			l.log.Error("the fallback client has tried all urls, but all failed")
			break
		}
		err := l.switchCurrentRpcLogic()
		if err != nil {
			l.log.Warn("the fallback client failed to switch the current client", "err", err)
		} else {
			break
		}
	}
}

func (l *FallbackClient) switchCurrentRpcLogic() error {
	url := l.urlList[l.currentIndex]
	newRpc, err := l.rpcInitFunc(url)
	if err != nil {
		return fmt.Errorf("the fallback client init RPC failed,url:%s, err:%v", url, err)
	}
	vErr := l.validateRpc(newRpc)
	if vErr != nil {
		return vErr
	}
	lastRpc := *l.currentRpc.Load()
	l.currentRpc.Store(&newRpc)
	if lastRpc != l.firstRpc {
		lastRpc.Close()
	}
	l.lastMinuteFail.Store(0)
	if l.subscribeFunc != nil {
		err := l.reSubscribeNewRpc(url)
		if err != nil {
			return err
		}
	}
	l.log.Info("switched current rpc to new url", "url", url)
	if !l.isInFallbackState {
		l.isInFallbackState = true
		l.recoverIfFirstRpcHealth()
	}
	return nil
}

func (l *FallbackClient) reSubscribeNewRpc(url string) error {
	(*l.headsSub).Unsubscribe()
	subscriptionNew, err := l.subscribeFunc()
	if err != nil {
		l.log.Error("can not subscribe new url", "url", url, "err", err)
		return err
	} else {
		*l.headsSub = subscriptionNew
	}
	return nil
}

func (l *FallbackClient) recoverIfFirstRpcHealth() {
	go func() {
		count := 0
		for {
			var id hexutil.Big
			err := l.firstRpc.CallContext(l.ctx, &id, "eth_chainId")
			if err != nil {
				count = 0
				time.Sleep(3 * time.Second)
				continue
			}
			count++
			if count >= 3 {
				break
			}
		}
		l.mx.Lock()
		defer l.mx.Unlock()
		if !l.isInFallbackState {
			return
		}
		lastRpc := *l.currentRpc.Load()
		l.currentRpc.Store(&l.firstRpc)
		lastRpc.Close()
		l.lastMinuteFail.Store(0)
		l.currentIndex = 0
		l.isInFallbackState = false
		if l.subscribeFunc != nil {
			err := l.reSubscribeNewRpc(l.urlList[0])
			if err != nil {
				l.log.Error("can not subscribe new url", "url", l.urlList[0], "err", err)
			}
		}
		l.log.Info("recover the current rpc to the first rpc", "url", l.urlList[0])
	}()
}

func (l *FallbackClient) RegisterSubscribeFunc(f func() (event.Subscription, error), headsSub *ethereum.Subscription) {
	l.subscribeFunc = f
	l.headsSub = headsSub
}

func (l *FallbackClient) validateRpc(newRpc client.RPC) error {
	chainID, err := l.ChainID(l.ctx, newRpc)
	if err != nil {
		return err
	}
	if l.chainId.Cmp(chainID) != 0 {
		return fmt.Errorf("incorrect RPC chain id %d, expected %d", chainID, l.chainId)
	}
	genesisBlockRef, err := l.blockRefByNumber(l.ctx, l.genesisBlock.Number, newRpc)
	if err != nil {
		return err
	}
	if genesisBlockRef.Hash != l.genesisBlock.Hash {
		return fmt.Errorf("incorrect genesis block hash %s, expected %s", genesisBlockRef.Hash, l.genesisBlock.Hash)
	}
	return nil
}

func (l *FallbackClient) ChainID(ctx context.Context, rpc client.RPC) (*big.Int, error) {
	var id hexutil.Big
	err := rpc.CallContext(ctx, &id, "eth_chainId")
	if err != nil {
		return nil, err
	}
	return (*big.Int)(&id), nil
}

func (l *FallbackClient) blockRefByNumber(ctx context.Context, number uint64, newRpc client.RPC) (*rpcHeader, error) {
	var header *rpcHeader
	err := newRpc.CallContext(ctx, &header, "eth_getBlockByNumber", numberID(number).Arg(), false) // headers are just blocks without txs
	if err != nil {
		return nil, err
	}
	return header, nil
}

func (l *FallbackClient) RegisterMetrics(metrics metrics.Metricer) {
	l.metrics = metrics
}

func (l *FallbackClient) GetCurrentIndex() int {
	return l.currentIndex
}
