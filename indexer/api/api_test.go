package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDB struct {
	mock.Mock
}

func (db *MockDB) GetDeposits(limit int, cursor string, sortDirection string) ([]api.Deposit, string, bool, error) {
	args := db.Called(limit, cursor, sortDirection)
	return args.Get(0).([]api.Deposit), args.String(1), args.Bool(2), args.Error(3)
}

func (db *MockDB) GetWithdrawals(limit int, cursor string, sortDirection string, sortBy string) ([]api.Withdrawal, string, bool, error) {
	args := db.Called(limit, cursor, sortDirection, sortBy)
	return args.Get(0).([]api.Withdrawal), args.String(1), args.Bool(2), args.Error(3)
}

func TestApi(t *testing.T) {
	mockDB := new(MockDB)

	mockDeposits := []api.Deposit{
		{
			Guid:            "test-guid",
			Amount:          "1000",
			BlockNumber:     123,
			BlockTimestamp:  time.Unix(123456, 0),
			From:            "0x1",
			To:              "0x2",
			TransactionHash: "0x3",
		},
	}

	mockWithdrawals := []api.Withdrawal{
		{
			Guid:            "test-guid",
			Amount:          "1000",
			BlockNumber:     123,
			BlockTimestamp:  time.Unix(123456, 0),
			From:            "0x1",
			To:              "0x2",
			TransactionHash: "0x3",
		},
	}

	mockDB.On("GetDeposits", 10, "", "").Return(mockDeposits, "nextCursor", false, nil)

	mockDB.On("GetWithdrawals", 10, "", "", "").Return(mockWithdrawals, "nextCursor", false, nil)

	testApi := api.NewApi(mockDB, mockDB)

	req, _ := http.NewRequest("GET", "/api/v0/deposits", nil)
	rr := httptest.NewRecorder()
	testApi.Router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "status code should be 200")

	// TODO make this type exist
	var depositsResponse api.DepositsResponse
	err := json.Unmarshal(rr.Body.Bytes(), &depositsResponse)
	assert.NoError(t, err)

	assert.Equal(t, mockDeposits, depositsResponse.Data)

	req, _ = http.NewRequest("GET", "/api/v0/withdrawals", nil)
	rr = httptest.NewRecorder()
	testApi.Router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "status code should be 200")

	// TODO make this type exist
	var withdrawalsResponse WithdrawalsResponse
	err = json
	err = json.Unmarshal(rr.Body.Bytes(), &withdrawalsResponse)
	assert.NoError(t, err)

	// Assert response data
	assert.Equal(t, mockWithdrawals, withdrawalsResponse.Data)

	// Finally, assert that the methods were called with the expected parameters
	mockDB.AssertCalled(t, "GetDeposits", 10, "", "")
	mockDB.AssertCalled(t, "GetWithdrawals", 10, "", "", "")
}
