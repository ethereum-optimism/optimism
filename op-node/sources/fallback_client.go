package sources

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type FallbackClientMetricer interface {
	RecordUrlSwitchEvt(url string)
}

type FallbackClientMetrics struct {
	urlSwitchEvt opmetrics.EventVec
}

func (f *FallbackClientMetrics) RecordUrlSwitchEvt(url string) {
	f.urlSwitchEvt.Record(url)
}

func NewFallbackClientMetrics(ns string, factory opmetrics.Factory) *FallbackClientMetrics {
	return &FallbackClientMetrics{
		urlSwitchEvt: opmetrics.NewEventVec(factory, ns, "", "url_switch", "url switch", []string{"url_idx"}),
	}
}

// FallbackClient is an RPC client, it can automatically switch to the next endpoint
// when there is a problem with the current endpoint
// and automatically switch back after the first endpoint recovers.
type FallbackClient struct {
	// firstRpc is created by the first of the urls, it should be used first in a healthy state
	firstRpc    client.RPC
	urlList     []string
	rpcInitFunc func(url string) (client.RPC, error)
	// lastMinuteFail is used to record the number of errors in the last minute
	lastMinuteFail atomic.Int64
	// currentRpc always points to the rpc currently being used
	currentRpc atomic.Pointer[client.RPC]
	// currentIndex is the index of the current rpc in the urlList
	currentIndex int
	mx           sync.Mutex
	log          log.Logger
	// isInFallbackState is used to record whether the current rpc is in a fallback state,
	// Used to ensure that only one recoverIfFirstRpcHealth process is started at the same time
	isInFallbackState bool
	// subscribeFunc is used when switching to an alternative rpc, we need to re-subscribe to the content subscribed by
	//the previous rpc, such as the subscription action in WatchHeadChanges, which needs to be triggered again
	subscribeFunc func() (event.Subscription, error)
	// headsSub is used to record the subscription of the previous rpc, so that we can unsubscribe when switching to
	// the new rpc
	headsSub *ethereum.Subscription
	// chainId and genesisBlock are used to check whether the newly switched rpc is legal.
	chainId      *big.Int
	genesisBlock eth.BlockID
	ctx          context.Context
	// isClose is used to close the goroutine that monitors the number of errors in the last minute
	isClose chan struct{}
	metrics FallbackClientMetricer
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
	// monitor the number of errors in the last minute
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
				// if the number of errors in the last minute exceeds the threshold, switch to the next rpc
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

// switchCurrentRpc switches to the next rpc
func (l *FallbackClient) switchCurrentRpc() {
	if l.currentIndex >= len(l.urlList) {
		l.log.Error("the fallback client has tried all urls, but all failed")
		return
	}
	l.mx.Lock()
	defer l.mx.Unlock()
	// double check to avoid switching to the next rpc at the same time
	if l.lastMinuteFail.Load() <= threshold {
		return
	}
	// iterate through the urlList to find the next available rpc
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
			if l.metrics != nil {
				l.metrics.RecordUrlSwitchEvt(strconv.Itoa(l.currentIndex))
			}
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
	// switch to the new rpc
	l.currentRpc.Store(&newRpc)
	// we don't close first rpc, the first rpc need to be recovered when it is healthy
	if lastRpc != l.firstRpc {
		lastRpc.Close()
	}
	// clear the number of errors in the last minute
	l.lastMinuteFail.Store(0)
	if l.subscribeFunc != nil {
		err := l.reSubscribeNewRpc(url)
		if err != nil {
			return err
		}
	}
	l.log.Info("switched current rpc to new url", "url", url)
	// start the process of recovering the first rpc if it has not been started
	if !l.isInFallbackState {
		l.isInFallbackState = true
		l.recoverIfFirstRpcHealth()
	}
	return nil
}

// reSubscribeNewRpc re-subscribe to the content subscribed by the previous rpc and unsubscribe the previous rpc
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

// recoverIfFirstRpcHealth recovers the first rpc if it is healthy
func (l *FallbackClient) recoverIfFirstRpcHealth() {
	go func() {
		count := 0
		for {
			var id hexutil.Big
			// use eth_chainId to check whether the first rpc is healthy
			err := l.firstRpc.CallContext(l.ctx, &id, "eth_chainId")
			if err != nil {
				count = 0
				time.Sleep(3 * time.Second)
				continue
			}
			count++
			// rpc is considered healthy if it succeeds in 3 consecutive requests.
			if count >= 3 {
				break
			}
		}
		// lock to avoid switching to the next rpc at the same time
		l.mx.Lock()
		defer l.mx.Unlock()
		// double check
		if !l.isInFallbackState {
			return
		}
		lastRpc := *l.currentRpc.Load()
		l.currentRpc.Store(&l.firstRpc)
		lastRpc.Close()
		l.lastMinuteFail.Store(0)
		// go back to the first index
		l.currentIndex = 0
		l.isInFallbackState = false
		// re-subscribe to the content subscribed by the previous rpc
		if l.subscribeFunc != nil {
			err := l.reSubscribeNewRpc(l.urlList[0])
			if err != nil {
				l.log.Error("can not subscribe new url", "url", l.urlList[0], "err", err)
			}
		}
		l.log.Info("recover the current rpc to the first rpc", "url", l.urlList[0])
	}()
}

// RegisterSubscribeFunc registers the function to be called when switching to the next rpc. It is not in the New
// function because this process and creating the fallback client are in two different code locations.
func (l *FallbackClient) RegisterSubscribeFunc(f func() (event.Subscription, error), headsSub *ethereum.Subscription) {
	l.subscribeFunc = f
	l.headsSub = headsSub
}

// validateRpc checks whether the newly switched rpc is legal.
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

// RegisterMetrics registers the metricer to record the url switch event
func (l *FallbackClient) RegisterMetrics(metrics FallbackClientMetricer) {
	l.metrics = metrics
}

func (l *FallbackClient) GetCurrentIndex() int {
	return l.currentIndex
}
