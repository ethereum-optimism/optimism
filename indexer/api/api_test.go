package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// MockBridgeView mocks the BridgeView interface
type MockBridgeView struct{}

// DepositsByAddress mocks returning deposits by an address
func (mbv *MockBridgeView) DepositsByAddress(address common.Address) ([]*database.DepositWithTransactionHash, error) {
	return []*database.DepositWithTransactionHash{
		{
			Deposit: database.Deposit{
				GUID:                 "mockGUID1",
				InitiatedL1EventGUID: "mockEventGUID1",
				Tx:                   database.Transaction{},
				TokenPair:            database.TokenPair{},
			},
			L1TransactionHash: common.HexToHash("0x123"),
		},
	}, nil
}

// WithdrawalsByAddress mocks returning withdrawals by an address
func (mbv *MockBridgeView) WithdrawalsByAddress(address common.Address) ([]*database.WithdrawalWithTransactionHashes, error) {
	return []*database.WithdrawalWithTransactionHashes{
		{
			Withdrawal: database.Withdrawal{
				GUID:                 "mockGUID2",
				InitiatedL2EventGUID: "mockEventGUID2",
				WithdrawalHash:       common.HexToHash("0x456"),
				Tx:                   database.Transaction{},
				TokenPair:            database.TokenPair{},
			},
			L2TransactionHash: common.HexToHash("0x789"),
		},
	}, nil
}

func TestHealthz(t *testing.T) {
	api := NewApi(&MockBridgeView{})
	request, err := http.NewRequest("GET", "/healthz", nil)
	assert.Nil(t, err)

	responseRecorder := httptest.NewRecorder()
	api.Router.ServeHTTP(responseRecorder, request)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
}

func TestDepositsHandler(t *testing.T) {
	api := NewApi(&MockBridgeView{})
	request, err := http.NewRequest("GET", "/api/v0/deposits/0x123", nil)
	assert.Nil(t, err)

	responseRecorder := httptest.NewRecorder()
	api.Router.ServeHTTP(responseRecorder, request)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
}

func TestWithdrawalsHandler(t *testing.T) {
	api := NewApi(&MockBridgeView{})
	request, err := http.NewRequest("GET", "/api/v0/withdrawals/0x123", nil)
	assert.Nil(t, err)

	responseRecorder := httptest.NewRecorder()
	api.Router.ServeHTTP(responseRecorder, request)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
}
