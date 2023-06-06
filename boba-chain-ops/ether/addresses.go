package ether

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/bindings"
	"github.com/bobanetwork/v3-anchorage/boba-bindings/predeploys"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/node"
	ethereum "github.com/ledgerwatch/erigon"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/log/v3"
)

type Crawler struct {
	Client             node.RPC
	EndBlock           int64
	RpcPollingInterval time.Duration
	OutputPath         string
	ctx                context.Context
	stop               chan struct{}
}

type EthAddresses struct {
	BlockNumber int64             `json:"blockNumber"`
	Addresses   []*common.Address `json:"addresses"`
}

func NewCrawler(client node.RPC, endBlock int64, rpcPollingInterval time.Duration, outputPath string) *Crawler {
	return &Crawler{
		Client:             client,
		EndBlock:           endBlock,
		RpcPollingInterval: rpcPollingInterval,
		OutputPath:         outputPath,
		ctx:                context.Background(),
		stop:               make(chan struct{}),
	}
}

func (e *Crawler) Start() error {
	currentBlock, addresses, err := e.LoadAddresses()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// insert the constant addresses to the addresses list
	addresses = append(addresses, &predeploys.SequencerFeeVaultAddr)

	go e.Loop(currentBlock, addresses)
	return nil
}

func (e *Crawler) Stop() {
	close(e.stop)
}

func (e *Crawler) Wait() {
	<-e.stop
}

func (e *Crawler) Loop(currentBlock int64, addresses []*common.Address) error {
	var err error
	timer := time.NewTicker(e.RpcPollingInterval)
	defer timer.Stop()
	mapAddresses := MapAddresses(addresses)
	for {
		select {
		case <-timer.C:
			currentBlock, mapAddresses, err = e.StartCrawler(currentBlock, mapAddresses)
			if err != nil {
				log.Error("error in crawler", "error", err)
			}
		case <-e.ctx.Done():
			e.Stop()
		}
	}
}

func (e *Crawler) StartCrawler(currentBlock int64, mapAddresses map[common.Address]bool) (int64, map[common.Address]bool, error) {
	var (
		err      error
		endBlock = big.NewInt(e.EndBlock)
	)
	if endBlock.Int64() == currentBlock {
		e.ctx.Done()
	}
	if endBlock.Cmp(common.Big0) == 0 {
		endBlock, err = e.Client.GetBlockNumber()
		if err != nil {
			return currentBlock, mapAddresses, err
		}
	}

	if currentBlock <= endBlock.Int64() {
		traceTransaction, err := e.GetTraceTransaction(big.NewInt(currentBlock))
		if err != nil {
			return currentBlock, mapAddresses, err
		}
		addresses, err := GetAddressesFromTrace(traceTransaction, true)
		if err != nil {
			return currentBlock, mapAddresses, err
		}
		mintAddress, err := e.GetToFromEthMintLogs(big.NewInt(currentBlock))
		if err != nil {
			return currentBlock, mapAddresses, err
		}
		log.Info("Crawled block", "block", currentBlock, "addresses", len(addresses), "mintAddress", len(mintAddress))
		addresses = append(addresses, mintAddress...)
		AddAddressesToMap(addresses, mapAddresses)
		e.SaveAddresses(currentBlock, MapToAddresses(mapAddresses))
		log.Info("Wrote addresses to file", "block", currentBlock, "addresses", len(mapAddresses))
		currentBlock++
	}

	return currentBlock, mapAddresses, nil
}

func (e *Crawler) LoadAddresses() (int64, []*common.Address, error) {
	file, err := os.Open(e.OutputPath)
	defer file.Close()
	if err != nil {
		return 1, nil, err
	}
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return 1, nil, err
	}
	var history EthAddresses
	if err := json.Unmarshal(byteValue, &history); err != nil {
		return 1, nil, err
	}
	return history.BlockNumber, history.Addresses, nil
}

func (e *Crawler) SaveAddresses(blockNumber int64, addresses []*common.Address) error {
	ethAddresses := EthAddresses{
		BlockNumber: blockNumber,
		Addresses:   addresses,
	}
	byteValue, err := json.Marshal(ethAddresses)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(e.OutputPath, byteValue, 0644); err != nil {
		return err
	}
	return nil
}

func (e *Crawler) GetTraceTransaction(blockNumber *big.Int) (*node.TraceTransaction, error) {
	block, err := e.Client.GetBlockByNumber(blockNumber)
	if err != nil {
		return nil, err
	}
	transactions := block.Transactions
	if len(transactions) == 0 {
		return nil, nil
	}
	if len(transactions) == 1 {
		transactionHash := transactions[0]
		traceTransaction, err := e.Client.TraceTransaction(transactionHash)
		if err != nil {
			return nil, err
		}
		return traceTransaction, nil
	}
	// This should not never happen on Boba legacy chain
	// more than one transaction in a block
	return nil, fmt.Errorf("block %d has more than one transaction", blockNumber)
}

func (e *Crawler) GetToFromEthMintLogs(blockNumber *big.Int) ([]*common.Address, error) {
	LegacyERC20ETHMetaData := bindings.MetaData{
		ABI: bindings.LegacyERC20ETHABI,
		Bin: bindings.LegacyERC20ETHBin,
	}
	ABI, err := LegacyERC20ETHMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	filter := ethereum.FilterQuery{
		FromBlock: blockNumber,
		ToBlock:   blockNumber,
		Addresses: []common.Address{
			*predeploys.Predeploys["LegacyERC20ETH"],
		},
	}
	var addresses []*common.Address
	logs, err := e.Client.GetLogs(&filter)
	if err != nil {
		return nil, err
	}
	for _, log := range logs {
		if log.Topics[0] == ABI.Events["Transfer"].ID && len(log.Topics) == 3 {
			to := common.BytesToAddress(log.Topics[2].Bytes())
			addresses = append(addresses, &to)
			// This case is for BOBA V1 that ETH can be transferred via
			// calling the OVM_ETH contract directly
			if log.Topics[1] != (common.Hash{}) {
				from := common.BytesToAddress(log.Topics[1].Bytes())
				addresses = append(addresses, &from)
			}
		}
	}
	return addresses, nil
}

func GetAddressesFromTrace(traceTransaction *node.TraceTransaction, sender bool) ([]*common.Address, error) {
	if traceTransaction == nil {
		return nil, nil
	}
	var addresses []*common.Address
	calls := traceTransaction
	value := calls.Value.ToInt()
	if sender {
		addresses = append(addresses, &calls.From)
	}
	if value.Cmp(common.Big0) == 1 {
		addresses = append(addresses, &calls.From)
		addresses = append(addresses, &calls.To)
	}
	if len(calls.Calls) == 0 {
		return addresses, nil
	}
	for _, call := range calls.Calls {
		innerAddress, err := GetAddressesFromTrace(call, false)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, innerAddress...)
	}
	return addresses, nil
}

func MapAddresses(addresses []*common.Address) map[common.Address]bool {
	addressMap := make(map[common.Address]bool)
	for _, address := range addresses {
		addressMap[*address] = true
	}
	return addressMap
}

func AddAddressesToMap(addresses []*common.Address, addressMap map[common.Address]bool) {
	for _, address := range addresses {
		addressMap[*address] = true
	}
}

func MapToAddresses(addressMap map[common.Address]bool) []*common.Address {
	var addresses []*common.Address
	for address := range addressMap {
		addr := address
		addresses = append(addresses, &addr)
	}
	return addresses
}

func CheckEthSlots(alloc types.GenesisAlloc, outputPath string) error {
	file, err := os.Open(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	bytes, _ := ioutil.ReadAll(file)
	if len(bytes) == 0 {
		return errors.New("Invalid eth addresses directory. The directory is empty.")
	}
	var addresses EthAddresses
	if err := json.Unmarshal(bytes, &addresses); err != nil {
		return err
	}

	validAddrCount := 0
	ethStorage := alloc[predeploys.LegacyERC20ETHAddr].Storage
	commonStorageKey := []common.Hash{
		common.BytesToHash([]byte{2}),
		common.BytesToHash([]byte{3}),
		common.BytesToHash([]byte{4}),
		common.BytesToHash([]byte{5}),
		common.BytesToHash([]byte{6}),
	}

	for _, addr := range addresses.Addresses {
		storageKey := CalcOVMETHStorageKey(*addr)
		if _, ok := ethStorage[storageKey]; ok {
			validAddrCount++
		}
	}
	for _, slot := range commonStorageKey {
		if _, ok := ethStorage[slot]; ok {
			validAddrCount++
		}
	}

	if len(ethStorage) != validAddrCount {
		log.Warn("Some addresses in eth addresses file are not valid", "valid", validAddrCount, "total", len(ethStorage))
		return fmt.Errorf("Some addresses in eth addresses file are not valid. Valid: %d, Total: %d", validAddrCount, len(ethStorage))
	}

	log.Info("All addresses in eth addresses file are valid", "valid", validAddrCount, "total", len(ethStorage))

	return nil
}
