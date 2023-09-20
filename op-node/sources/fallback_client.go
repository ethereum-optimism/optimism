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

// FallbackClient is an RPC client, it can automatically switch to the next l1 endpoint
// when there is a problem with the current l1 endpoint
// and automatically switch back after the first l1 endpoint recovers.
type FallbackClient struct {
	// firstRpc is created by the first of the l1 urls, it should be used first in a healthy state
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
	l1HeadsSub        *ethereum.Subscription
	l1ChainId         *big.Int
	l1Block           eth.BlockID
	ctx               context.Context
	isClose           chan struct{}
	metrics           metrics.Metricer
}

const threshold int64 = 20

// NewFallbackClient returns a new FallbackClient. l1ChainId and l1Block are used to check
// whether the newly switched rpc is legal.
func NewFallbackClient(ctx context.Context, rpc client.RPC, urlList []string, log log.Logger, l1ChainId *big.Int, l1Block eth.BlockID, rpcInitFunc func(url string) (client.RPC, error)) client.RPC {
	fallbackClient := &FallbackClient{
		ctx:          ctx,
		firstRpc:     rpc,
		urlList:      urlList,
		log:          log,
		rpcInitFunc:  rpcInitFunc,
		currentIndex: 0,
		l1ChainId:    l1ChainId,
		l1Block:      l1Block,
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
		l.metrics.RecordL1UrlSwitchEvent()
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
	(*l.l1HeadsSub).Unsubscribe()
	subscriptionNew, err := l.subscribeFunc()
	if err != nil {
		l.log.Error("can not subscribe new url", "url", url, "err", err)
		return err
	} else {
		*l.l1HeadsSub = subscriptionNew
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

func (l *FallbackClient) RegisterSubscribeFunc(f func() (event.Subscription, error), l1HeadsSub *ethereum.Subscription) {
	l.subscribeFunc = f
	l.l1HeadsSub = l1HeadsSub
}

func (l *FallbackClient) validateRpc(newRpc client.RPC) error {
	chainID, err := l.ChainID(l.ctx, newRpc)
	if err != nil {
		return err
	}
	if l.l1ChainId.Cmp(chainID) != 0 {
		return fmt.Errorf("incorrect L1 RPC chain id %d, expected %d", chainID, l.l1ChainId)
	}
	l1GenesisBlockRef, err := l.l1BlockRefByNumber(l.ctx, l.l1Block.Number, newRpc)
	if err != nil {
		return err
	}
	if l1GenesisBlockRef.Hash != l.l1Block.Hash {
		return fmt.Errorf("incorrect L1 genesis block hash %s, expected %s", l1GenesisBlockRef.Hash, l.l1Block.Hash)
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

func (l *FallbackClient) l1BlockRefByNumber(ctx context.Context, number uint64, newRpc client.RPC) (*rpcHeader, error) {
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
