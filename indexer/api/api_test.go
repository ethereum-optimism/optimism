package api

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockBridgeView mocks the BridgeView interface
type MockBridgeView struct{}

const (
	guid1 = "8408b6d2-7c90-4cfc-8604-b2204116cb6a"
	guid2 = "8408b6d2-7c90-4cfc-8604-b2204116cb6b"
)

// DepositsByAddress mocks returning deposits by an address
func (mbv *MockBridgeView) DepositsByAddress(address common.Address) ([]*database.DepositWithTransactionHashes, error) {
	return []*database.DepositWithTransactionHashes{
		{
			Deposit: database.Deposit{
				GUID:                 uuid.MustParse(guid1),
				InitiatedL1EventGUID: uuid.MustParse(guid2),
				Tx:                   database.Transaction{},
				TokenPair:            database.TokenPair{},
			},
			L1TransactionHash: common.HexToHash("0x123"),
		},
	}, nil
}

// DepositsByAddress mocks returning deposits by an address
func (mbv *MockBridgeView) DepositByMessageNonce(nonce *big.Int) (*database.Deposit, error) {
	return &database.Deposit{
		GUID:                 uuid.MustParse(guid1),
		InitiatedL1EventGUID: uuid.MustParse(guid2),
		Tx:                   database.Transaction{},
		TokenPair:            database.TokenPair{},
	}, nil
}

// LatestDepositMessageNonce mocks returning the latest cross domain message nonce for a deposit
func (mbv *MockBridgeView) LatestDepositMessageNonce() (*big.Int, error) {
	return big.NewInt(0), nil
}

// WithdrawalsByAddress mocks returning withdrawals by an address
func (mbv *MockBridgeView) WithdrawalsByAddress(address common.Address) ([]*database.WithdrawalWithTransactionHashes, error) {
	return []*database.WithdrawalWithTransactionHashes{
		{
			Withdrawal: database.Withdrawal{
				GUID:                 uuid.MustParse(guid2),
				InitiatedL2EventGUID: uuid.MustParse(guid1),
				WithdrawalHash:       common.HexToHash("0x456"),
				Tx:                   database.Transaction{},
				TokenPair:            database.TokenPair{},
			},
			L2TransactionHash: common.HexToHash("0x789"),
		},
	}, nil
}

// WithdrawalsByMessageNonce mocks returning withdrawals by a withdrawal hash
func (mbv *MockBridgeView) WithdrawalByMessageNonce(nonce *big.Int) (*database.Withdrawal, error) {
	return &database.Withdrawal{
		GUID:                 uuid.MustParse(guid2),
		InitiatedL2EventGUID: uuid.MustParse(guid1),
		WithdrawalHash:       common.HexToHash("0x456"),
		Tx:                   database.Transaction{},
		TokenPair:            database.TokenPair{},
	}, nil
}

// WithdrawalsByHash mocks returning withdrawals by a withdrawal hash
func (mbv *MockBridgeView) WithdrawalByHash(address common.Hash) (*database.Withdrawal, error) {
	return &database.Withdrawal{
		GUID:                 uuid.MustParse(guid2),
		InitiatedL2EventGUID: uuid.MustParse(guid1),
		WithdrawalHash:       common.HexToHash("0x456"),
		Tx:                   database.Transaction{},
		TokenPair:            database.TokenPair{},
	}, nil
}

// LatestWithdrawalMessageNonce mocks returning the latest cross domain message nonce for a withdrawal
func (mbv *MockBridgeView) LatestWithdrawalMessageNonce() (*big.Int, error) {
	return big.NewInt(0), nil
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
