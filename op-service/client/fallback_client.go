package client

import (
	"context"
	"errors"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

// FallbackClient is an EthClient, it can automatically switch to the next endpoint
// when there is a problem with the current endpoint
// and automatically switch back after the first endpoint recovers.
type FallbackClient struct {
	// firstClient is created by the first of the urls, it should be used first in a healthy state
	firstClient    EthClient
	urlList        []string
	clientInitFunc func(url string) (EthClient, error)
	// lastMinuteFail is used to record the number of errors in the last minute
	lastMinuteFail atomic.Int64
	// currentClient always points to the client currently being used
	currentClient atomic.Pointer[EthClient]
	// currentIndex is the index of the current client in the urlList
	currentIndex int
	mx           sync.Mutex
	log          log.Logger
	// isInFallbackState is used to record whether the current rpc is in a fallback state,
	// Used to ensure that only one recoverIfFirstClientHealth process is started at the same time
	isInFallbackState bool
	// fallbackThreshold specifies how many errors have occurred in the past 1 minute to trigger the switching logic
	fallbackThreshold int64
	// isClose is used to close the goroutine that monitors the number of errors in the last minute
	isClose chan struct{}
	metrics FallbackClientMetricer
}

// NewFallbackClient returns a new FallbackClient.
func NewFallbackClient(rpc EthClient, urlList []string, log log.Logger, fallbackThreshold int64, m FallbackClientMetricer, clientInitFunc func(url string) (EthClient, error)) EthClient {
	fallbackClient := &FallbackClient{
		firstClient:       rpc,
		urlList:           urlList,
		log:               log,
		clientInitFunc:    clientInitFunc,
		currentIndex:      0,
		fallbackThreshold: fallbackThreshold,
		metrics:           m,
	}
	fallbackClient.currentClient.Store(&rpc)
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
				// if the number of errors in the last minute exceeds the threshold, switch to the next client
				if fallbackClient.lastMinuteFail.Load() >= fallbackClient.fallbackThreshold {
					fallbackClient.switchCurrentClient()
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
	return fallbackClient
}

func (l *FallbackClient) BlockNumber(ctx context.Context) (uint64, error) {
	number, err := (*l.currentClient.Load()).BlockNumber(ctx)
	if err != nil {
		l.handleErr(err)
	}
	return number, err
}

func (l *FallbackClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	receipt, err := (*l.currentClient.Load()).TransactionReceipt(ctx, txHash)
	if err != nil {
		l.handleErr(err)
	}
	return receipt, err
}

func (l *FallbackClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	err := (*l.currentClient.Load()).SendTransaction(ctx, tx)
	if err != nil {
		l.handleErr(err)
	}
	return err
}

func (l *FallbackClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	tipCap, err := (*l.currentClient.Load()).SuggestGasTipCap(ctx)
	if err != nil {
		l.handleErr(err)
	}
	return tipCap, err
}

func (l *FallbackClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	at, err := (*l.currentClient.Load()).PendingNonceAt(ctx, account)
	if err != nil {
		l.handleErr(err)
	}
	return at, err
}

func (l *FallbackClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	estimateGas, err := (*l.currentClient.Load()).EstimateGas(ctx, msg)
	if err != nil {
		l.handleErr(err)
	}
	return estimateGas, err
}

func (l *FallbackClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	contract, err := (*l.currentClient.Load()).CallContract(ctx, call, blockNumber)
	if err != nil {
		l.handleErr(err)
	}
	return contract, err
}

func (l *FallbackClient) Close() {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.isClose <- struct{}{}
	currentClient := *l.currentClient.Load()
	currentClient.Close()
	if currentClient != l.firstClient {
		l.firstClient.Close()
	}
}

func (l *FallbackClient) ChainID(ctx context.Context) (*big.Int, error) {
	id, err := (*l.currentClient.Load()).ChainID(ctx)
	if err != nil {
		l.handleErr(err)
	}
	return id, err
}

func (l *FallbackClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	balanceAt, err := (*l.currentClient.Load()).BalanceAt(ctx, account, blockNumber)
	if err != nil {
		l.handleErr(err)
	}
	return balanceAt, err
}

func (l *FallbackClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	headerByNumber, err := (*l.currentClient.Load()).HeaderByNumber(ctx, number)
	if err != nil {
		l.handleErr(err)
	}
	return headerByNumber, err
}

func (l *FallbackClient) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	storageAt, err := (*l.currentClient.Load()).StorageAt(ctx, account, key, blockNumber)
	if err != nil {
		l.handleErr(err)
	}
	return storageAt, err
}

func (l *FallbackClient) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	codeAt, err := (*l.currentClient.Load()).CodeAt(ctx, account, blockNumber)
	if err != nil {
		l.handleErr(err)
	}
	return codeAt, err
}

func (l *FallbackClient) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	nonceAt, err := (*l.currentClient.Load()).NonceAt(ctx, account, blockNumber)
	if err != nil {
		l.handleErr(err)
	}
	return nonceAt, err
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

// switchCurrentClient switches the current client to the next client
func (l *FallbackClient) switchCurrentClient() {
	l.mx.Lock()
	defer l.mx.Unlock()
	// double check to avoid switching to the next client at the same time
	if l.lastMinuteFail.Load() <= l.fallbackThreshold {
		return
	}
	// iterate through the urlList to find the next available rpc
	for {
		l.currentIndex++
		if l.currentIndex >= len(l.urlList) {
			l.log.Error("the fallback client has tried all urls, but all failed")
			break
		}
		err := l.switchCurrentClientLogic()
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

func (l *FallbackClient) switchCurrentClientLogic() error {
	url := l.urlList[l.currentIndex]
	newClient, err := l.clientInitFunc(url)
	if err != nil {
		l.log.Error("the fallback client failed to switch the current client", "url", url, "err", err)
		return err
	}
	lastClient := *l.currentClient.Load()
	// switch to the new client
	l.currentClient.Store(&newClient)
	// we don't close first client, the first client need to be recovered when it is healthy
	if lastClient != l.firstClient {
		lastClient.Close()
	}
	// clear the number of errors in the last minute
	l.lastMinuteFail.Store(0)
	l.log.Info("switched current client to new url", "url", url)
	if !l.isInFallbackState {
		l.isInFallbackState = true
		l.recoverIfFirstClientHealth()
	}
	return nil
}

// recoverIfFirstClientHealth recovers the first client if it is healthy
func (l *FallbackClient) recoverIfFirstClientHealth() {
	go func() {
		count := 0
		for {
			// use eth_chainId to check whether the first client is healthy
			_, err := l.firstClient.ChainID(context.Background())
			if err != nil {
				count = 0
				time.Sleep(3 * time.Second)
				continue
			}
			count++
			// the client is considered healthy if it succeeds in 3 consecutive requests.
			if count >= 3 {
				break
			}
		}
		// lock to avoid switching to the next client at the same time
		l.mx.Lock()
		defer l.mx.Unlock()
		// double check
		if !l.isInFallbackState {
			return
		}
		lastClient := *l.currentClient.Load()
		l.currentClient.Store(&l.firstClient)
		lastClient.Close()
		l.lastMinuteFail.Store(0)
		// go back to the first index
		l.currentIndex = 0
		l.isInFallbackState = false
		l.log.Info("recover the current client to the first client", "url", l.urlList[0])
	}()
}
