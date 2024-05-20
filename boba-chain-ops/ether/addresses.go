package ether

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"time"

	"github.com/bobanetwork/boba/boba-bindings/bindings"
	"github.com/bobanetwork/boba/boba-bindings/predeploys"
	"github.com/bobanetwork/boba/boba-chain-ops/node"
	ethereum "github.com/ledgerwatch/erigon"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/log/v3"
)

type Crawler struct {
	Client             node.RPC
	BackupClient       node.RPC
	EndBlock           int64
	RpcPollingInterval time.Duration
	AddrOutputPath     string
	AlloOutputPath     string
	ctx                context.Context
	stop               chan struct{}
}

type EthAddresses struct {
	BlockNumber int64             `json:"blockNumber"`
	Addresses   []*common.Address `json:"addresses"`
}

// Allowance represents the allowances that were set in the
// legacy ERC20 representation of ether
type Allowance struct {
	From common.Address `json:"fr"`
	To   common.Address `json:"to"`
}

type EthAllowances struct {
	BlockNumber int64        `json:"blockNumber"`
	Allowances  []*Allowance `json:"allowances"`
}

func NewCrawler(client node.RPC, backupClient node.RPC, endBlock int64, rpcPollingInterval time.Duration, addrOutputPath, alloOutputPath string) *Crawler {
	return &Crawler{
		Client:             client,
		BackupClient:       backupClient,
		EndBlock:           endBlock,
		RpcPollingInterval: rpcPollingInterval,
		AddrOutputPath:     addrOutputPath,
		AlloOutputPath:     alloOutputPath,
		ctx:                context.Background(),
		stop:               make(chan struct{}),
	}
}

func (e *Crawler) Start() error {
	addrCurrentBlock, addresses, err := e.LoadAddresses()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// insert the constant addresses to the addresses list
	addresses = append(addresses, &predeploys.SequencerFeeVaultAddr)

	alloCurrentBlock, allowances, err := e.LoadAllowances()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	currentBlock := addrCurrentBlock
	if alloCurrentBlock < currentBlock {
		currentBlock = alloCurrentBlock
	}
	go e.Loop(currentBlock, addresses, allowances)
	return nil
}

func (e *Crawler) Stop() {
	close(e.stop)
}

func (e *Crawler) Wait() {
	<-e.stop
}

func (e *Crawler) Loop(currentBlock int64, addresses []*common.Address, allowances []*Allowance) {
	var err error
	timer := time.NewTicker(e.RpcPollingInterval)
	defer timer.Stop()
	mapAddresses := MapAddresses(addresses)
	mapAllowances := MapAllowances(allowances)
	for {
		select {
		case <-timer.C:
			currentBlock, mapAddresses, mapAllowances, err = e.StartCrawler(currentBlock, mapAddresses, mapAllowances)
			if err != nil {
				log.Error("error in crawler", "error", err)
			}
		case <-e.ctx.Done():
			e.Stop()
		}
	}
}

func (e *Crawler) StartCrawler(currentBlock int64, mapAddresses map[common.Address]bool, mapAllowances map[Allowance]bool) (int64, map[common.Address]bool, map[Allowance]bool, error) {
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
			return currentBlock, mapAddresses, mapAllowances, err
		}
	}

	if currentBlock <= endBlock.Int64() {
		traceTransaction, err := e.GetTraceTransaction(big.NewInt(currentBlock))
		if err != nil {
			return currentBlock, mapAddresses, mapAllowances, err
		}
		addresses, err := GetAddressesFromTrace(traceTransaction, true)
		if err != nil {
			return currentBlock, mapAddresses, mapAllowances, err
		}
		ethAddresses, ethAllowances, err := e.GetToFromEthLogs(big.NewInt(currentBlock))
		if err != nil {
			return currentBlock, mapAddresses, mapAllowances, err
		}

		log.Info("Crawled block", "block", currentBlock, "addresses", len(addresses), "transfer log", len(ethAddresses), "allowance log", len(ethAllowances))

		addresses = append(addresses, ethAddresses...)
		AddAddressesToMap(addresses, mapAddresses)
		if err := e.SaveAddresses(currentBlock, MapToAddresses(mapAddresses)); err != nil {
			return currentBlock, mapAddresses, mapAllowances, err
		}

		AddAllowancesToMap(mapAllowances, MapAllowances(ethAllowances))
		if err := e.SaveAllowances(currentBlock, MapToAllowances(mapAllowances)); err != nil {
			return currentBlock, mapAddresses, mapAllowances, err
		}

		log.Info("Wrote addresses and allowances to file", "block", currentBlock, "addresses", len(mapAddresses), "allowances", len(mapAllowances))
		currentBlock++
	}

	return currentBlock, mapAddresses, mapAllowances, nil
}

func (e *Crawler) LoadAddresses() (int64, []*common.Address, error) {
	file, err := os.Open(e.AddrOutputPath)
	if err != nil {
		return 1, nil, err
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return 1, nil, err
	}
	var history EthAddresses
	if err := json.Unmarshal(byteValue, &history); err != nil {
		return 1, nil, err
	}
	return history.BlockNumber, history.Addresses, nil
}

func (e *Crawler) LoadAllowances() (int64, []*Allowance, error) {
	file, err := os.Open(e.AlloOutputPath)
	if err != nil {
		return 1, nil, err
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return 1, nil, err
	}
	var history EthAllowances
	if err := json.Unmarshal(byteValue, &history); err != nil {
		return 1, nil, err
	}
	return history.BlockNumber, history.Allowances, nil
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
	f, err := os.OpenFile(e.AddrOutputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(byteValue); err != nil {
		return err
	}
	return nil
}

func (e *Crawler) SaveAllowances(blockNumber int64, allowances []*Allowance) error {
	ethAllowances := EthAllowances{
		BlockNumber: blockNumber,
		Allowances:  allowances,
	}
	byteValue, err := json.Marshal(ethAllowances)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(e.AlloOutputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(byteValue); err != nil {
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
			if e.BackupClient != nil {
				traceTransaction, err := e.BackupClient.TraceTransaction(transactionHash)
				if err != nil {
					return nil, fmt.Errorf("failed to trace transaction from backup RPC %s: %w", transactionHash.String(), err)
				}
				return traceTransaction, nil
			}
			return nil, fmt.Errorf("failed to trace transaction %s: %w", transactionHash.String(), err)
		}
		return traceTransaction, nil
	}
	// This should not never happen on Boba legacy chain
	// more than one transaction in a block
	return nil, fmt.Errorf("block %d has more than one transaction", blockNumber)
}

func (e *Crawler) GetToFromEthLogs(blockNumber *big.Int) ([]*common.Address, []*Allowance, error) {
	LegacyERC20ETHMetaData := bindings.MetaData{
		ABI: bindings.LegacyERC20ETHABI,
		Bin: bindings.LegacyERC20ETHBin,
	}
	ABI, err := LegacyERC20ETHMetaData.GetAbi()
	if err != nil {
		return nil, nil, err
	}
	filter := ethereum.FilterQuery{
		FromBlock: blockNumber,
		ToBlock:   blockNumber,
		Addresses: []common.Address{
			*predeploys.Predeploys["LegacyERC20ETH"],
		},
	}
	var addresses []*common.Address
	var allowances []*Allowance
	logs, err := e.Client.GetLogs(&filter)
	if err != nil {
		return nil, nil, err
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
		// This case for BOBA V1 that ETH can be approved via calling
		// the OVM_ETH contract directly
		if log.Topics[0] == ABI.Events["Approval"].ID && len(log.Topics) == 3 {
			owner := common.BytesToAddress(log.Topics[1].Bytes())
			spender := common.BytesToAddress(log.Topics[2].Bytes())
			allowances = append(allowances, &Allowance{
				From: owner,
				To:   spender,
			})
		}
	}
	return addresses, allowances, nil
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

func MapToAddresses(addressesMap map[common.Address]bool) []*common.Address {
	var addresses []*common.Address
	for address := range addressesMap {
		addr := address
		addresses = append(addresses, &addr)
	}
	return addresses
}

func MapAllowances(allowances []*Allowance) map[Allowance]bool {
	allowanceMap := make(map[Allowance]bool)
	for _, allowance := range allowances {
		allowanceMap[*allowance] = true
	}
	return allowanceMap
}

func AddAllowancesToMap(allowances map[Allowance]bool, allowancesMap map[Allowance]bool) {
	for allowance, _ := range allowancesMap {
		allowances[allowance] = true
	}
}

func MapToAllowances(allowancesMap map[Allowance]bool) []*Allowance {
	var allowances []*Allowance
	for allowance := range allowancesMap {
		allo := allowance
		allowances = append(allowances, &allo)
	}
	return allowances
}

func CheckEthSlots(alloc types.GenesisAlloc, addrOutputPath, alloOutputPath string) error {
	addrFile, err := os.Open(addrOutputPath)
	if err != nil {
		return err
	}
	defer addrFile.Close()
	addBytes, _ := io.ReadAll(addrFile)
	if len(addBytes) == 0 {
		return errors.New("Invalid eth addresses directory. The directory is empty.")
	}
	var addresses EthAddresses
	if err := json.Unmarshal(addBytes, &addresses); err != nil {
		return err
	}

	alloFile, err := os.Open(alloOutputPath)
	if err != nil {
		return err
	}
	defer alloFile.Close()
	alloBytes, _ := io.ReadAll(alloFile)
	if len(alloBytes) == 0 {
		return errors.New("Invalid eth allowances directory. The directory is empty.")
	}
	var allowances EthAllowances
	if err := json.Unmarshal(alloBytes, &allowances); err != nil {
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
	for _, allo := range allowances.Allowances {
		storageKey := CalcAllowanceStorageKey(allo.From, allo.To)
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
		log.Warn("Some addresses in eth addresses and eth allowances files are not valid", "valid", validAddrCount, "total", len(ethStorage))
		return fmt.Errorf("Some addresses in eeth addresses and eth allowances files are not valid. Valid: %d, Total: %d", validAddrCount, len(ethStorage))
	}

	log.Info("All addresses in eth addresses and eth allowances files are valid", "valid", validAddrCount, "total", len(ethStorage))

	return nil
}
