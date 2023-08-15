package api

import (
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
)

// MockBridgeTransfersView mocks the BridgeTransfersView interface
type MockBridgeTransfersView struct{}

var mockAddress = "0x4204204204204204204204204204204204204204"

var (
	deposit = database.L1BridgeDeposit{
		TransactionSourceHash:     common.HexToHash("abc"),
		CrossDomainMessengerNonce: &database.U256{Int: big.NewInt(0)},
		Tx:                        database.Transaction{},
		TokenPair:                 database.TokenPair{},
	}

	withdrawal = database.L2BridgeWithdrawal{
		TransactionWithdrawalHash: common.HexToHash("0x420"),
		CrossDomainMessengerNonce: &database.U256{Int: big.NewInt(0)},
		Tx:                        database.Transaction{},
		TokenPair:                 database.TokenPair{},
	}
)

func (mbv *MockBridgeTransfersView) L1BridgeDeposit(hash common.Hash) (*database.L1BridgeDeposit, error) {
	return &deposit, nil
}

func (mbv *MockBridgeTransfersView) L1BridgeDepositByCrossDomainMessengerNonce(nonce *big.Int) (*database.L1BridgeDeposit, error) {
	return &deposit, nil
}

func (mbv *MockBridgeTransfersView) L1BridgeDepositsByAddress(address common.Address) ([]*database.L1BridgeDepositWithTransactionHashes, error) {
	return []*database.L1BridgeDepositWithTransactionHashes{
		{
			L1BridgeDeposit:   deposit,
			L1TransactionHash: common.HexToHash("0x123"),
		},
	}, nil
}

func (mbv *MockBridgeTransfersView) L2BridgeWithdrawal(address common.Hash) (*database.L2BridgeWithdrawal, error) {
	return &withdrawal, nil
}

func (mbv *MockBridgeTransfersView) L2BridgeWithdrawalByCrossDomainMessengerNonce(nonce *big.Int) (*database.L2BridgeWithdrawal, error) {
	return &withdrawal, nil
}

func (mbv *MockBridgeTransfersView) L2BridgeWithdrawalsByAddress(address common.Address) ([]*database.L2BridgeWithdrawalWithTransactionHashes, error) {
	return []*database.L2BridgeWithdrawalWithTransactionHashes{
		{
			L2BridgeWithdrawal: withdrawal,
			L2TransactionHash:  common.HexToHash("0x789"),
		},
	}, nil
}

func TestHealthz(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	api := NewApi(&MockBridgeTransfersView{}, logger)
	request, err := http.NewRequest("GET", "/healthz", nil)
	assert.Nil(t, err)

	responseRecorder := httptest.NewRecorder()
	api.Router.ServeHTTP(responseRecorder, request)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
}

func TestL1BridgeDepositsHandler(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	api := NewApi(&MockBridgeTransfersView{}, logger)
	request, err := http.NewRequest("GET", fmt.Sprintf("/api/v0/deposits/%s", mockAddress), nil)
	assert.Nil(t, err)

	responseRecorder := httptest.NewRecorder()
	api.Router.ServeHTTP(responseRecorder, request)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
}

func TestL2BridgeWithdrawalsByAddressHandler(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	api := NewApi(&MockBridgeTransfersView{}, logger)
	request, err := http.NewRequest("GET", fmt.Sprintf("/api/v0/withdrawals/%s", mockAddress), nil)
	assert.Nil(t, err)

	responseRecorder := httptest.NewRecorder()
	api.Router.ServeHTTP(responseRecorder, request)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
}
