package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/api/routes"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBridgeTransfersView mocks the BridgeTransfersView interface
type MockBridgeTransfersView struct{}

var mockAddress = "0x4204204204204204204204204204204204204204"

var apiConfig = config.ServerConfig{
	Host: "localhost",
	Port: 8080,
}
var metricsConfig = config.ServerConfig{
	Host: "localhost",
	Port: 7300,
}

var (
	deposit = database.L1BridgeDeposit{
		TransactionSourceHash: common.HexToHash("abc"),
		BridgeTransfer: database.BridgeTransfer{
			CrossDomainMessageHash: &common.Hash{},
			Tx:                     database.Transaction{},
			TokenPair:              database.TokenPair{},
		},
	}

	withdrawal = database.L2BridgeWithdrawal{
		TransactionWithdrawalHash: common.HexToHash("0x420"),
		BridgeTransfer: database.BridgeTransfer{
			CrossDomainMessageHash: &common.Hash{},
			Tx:                     database.Transaction{},
			TokenPair:              database.TokenPair{},
		},
	}
)

func (mbv *MockBridgeTransfersView) L1BridgeDeposit(hash common.Hash) (*database.L1BridgeDeposit, error) {
	return &deposit, nil
}

func (mbv *MockBridgeTransfersView) L1BridgeDepositWithFilter(filter database.BridgeTransfer) (*database.L1BridgeDeposit, error) {
	return &deposit, nil
}

func (mbv *MockBridgeTransfersView) L2BridgeWithdrawal(address common.Hash) (*database.L2BridgeWithdrawal, error) {
	return &withdrawal, nil
}

func (mbv *MockBridgeTransfersView) L2BridgeWithdrawalWithFilter(filter database.BridgeTransfer) (*database.L2BridgeWithdrawal, error) {
	return &withdrawal, nil
}

func (mbv *MockBridgeTransfersView) L1BridgeDepositsByAddress(address common.Address, cursor string, limit int) (*database.L1BridgeDepositsResponse, error) {
	return &database.L1BridgeDepositsResponse{
		Deposits: []database.L1BridgeDepositWithTransactionHashes{
			{
				L1BridgeDeposit:   deposit,
				L1TransactionHash: common.HexToHash("0x123"),
				L2TransactionHash: common.HexToHash("0x555"),
				L1BlockHash:       common.HexToHash("0x456"),
			},
		},
	}, nil
}

func (mbv *MockBridgeTransfersView) L2BridgeWithdrawalsByAddress(address common.Address, cursor string, limit int) (*database.L2BridgeWithdrawalsResponse, error) {
	return &database.L2BridgeWithdrawalsResponse{
		Withdrawals: []database.L2BridgeWithdrawalWithTransactionHashes{
			{
				L2BridgeWithdrawal:         withdrawal,
				L2TransactionHash:          common.HexToHash("0x789"),
				L2BlockHash:                common.HexToHash("0x456"),
				ProvenL1TransactionHash:    common.HexToHash("0x123"),
				FinalizedL1TransactionHash: common.HexToHash("0x123"),
			},
		},
	}, nil
}
func TestHealthz(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	api := NewApi(logger, &MockBridgeTransfersView{}, apiConfig, metricsConfig)
	request, err := http.NewRequest("GET", "/healthz", nil)
	assert.Nil(t, err)

	responseRecorder := httptest.NewRecorder()
	api.Router.ServeHTTP(responseRecorder, request)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)
}

func TestL1BridgeDepositsHandler(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	api := NewApi(logger, &MockBridgeTransfersView{}, apiConfig, metricsConfig)
	request, err := http.NewRequest("GET", fmt.Sprintf("/api/v0/deposits/%s", mockAddress), nil)
	assert.Nil(t, err)

	responseRecorder := httptest.NewRecorder()
	api.Router.ServeHTTP(responseRecorder, request)

	assert.Equal(t, http.StatusOK, responseRecorder.Code)

	var resp routes.DepositResponse
	err = json.Unmarshal(responseRecorder.Body.Bytes(), &resp)
	assert.Nil(t, err)

	require.Len(t, resp.Items, 1)

	assert.Equal(t, resp.Items[0].L1BlockHash, common.HexToHash("0x456").String())
	assert.Equal(t, resp.Items[0].L1TxHash, common.HexToHash("0x123").String())
	assert.Equal(t, resp.Items[0].Timestamp, deposit.Tx.Timestamp)
	assert.Equal(t, resp.Items[0].L2TxHash, common.HexToHash("555").String())
}

func TestL2BridgeWithdrawalsByAddressHandler(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	api := NewApi(logger, &MockBridgeTransfersView{}, apiConfig, metricsConfig)
	request, err := http.NewRequest("GET", fmt.Sprintf("/api/v0/withdrawals/%s", mockAddress), nil)
	assert.Nil(t, err)

	responseRecorder := httptest.NewRecorder()
	api.Router.ServeHTTP(responseRecorder, request)

	var resp routes.WithdrawalResponse
	err = json.Unmarshal(responseRecorder.Body.Bytes(), &resp)
	assert.Nil(t, err)

	require.Len(t, resp.Items, 1)

	assert.Equal(t, resp.Items[0].Guid, withdrawal.TransactionWithdrawalHash.String())
	assert.Equal(t, resp.Items[0].L2BlockHash, common.HexToHash("0x456").String())
	assert.Equal(t, resp.Items[0].From, withdrawal.Tx.FromAddress.String())
	assert.Equal(t, resp.Items[0].To, withdrawal.Tx.ToAddress.String())
	assert.Equal(t, resp.Items[0].TransactionHash, common.HexToHash("0x789").String())
	assert.Equal(t, resp.Items[0].Amount, withdrawal.Tx.Amount.String())
	assert.Equal(t, resp.Items[0].ProofTransactionHash, common.HexToHash("0x123").String())
	assert.Equal(t, resp.Items[0].ClaimTransactionHash, common.HexToHash("0x123").String())
	assert.Equal(t, resp.Items[0].L1TokenAddress, withdrawal.TokenPair.RemoteTokenAddress.String())
	assert.Equal(t, resp.Items[0].L2TokenAddress, withdrawal.TokenPair.LocalTokenAddress.String())
	assert.Equal(t, resp.Items[0].Timestamp, withdrawal.Tx.Timestamp)

}
