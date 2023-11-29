package service_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/api/service"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func assertFieldsAreSet(t *testing.T, item any) {
	structType := reflect.TypeOf(item)

	structVal := reflect.ValueOf(item)
	fieldNum := structVal.NumField()

	for i := 0; i < fieldNum; i++ {
		field := structVal.Field(i)
		fieldName := structType.Field(i).Name

		isSet := field.IsValid() && !field.IsZero()

		require.True(t, isSet, fmt.Sprintf("%s in not set", fieldName))

	}
}

func TestWithdrawalResponse(t *testing.T) {
	svc := service.New(nil, nil, nil)
	cdh := common.HexToHash("0x2")

	withdraws := &database.L2BridgeWithdrawalsResponse{
		Withdrawals: []database.L2BridgeWithdrawalWithTransactionHashes{
			{
				L2BridgeWithdrawal: database.L2BridgeWithdrawal{
					TransactionWithdrawalHash: common.HexToHash("0x1"),
					BridgeTransfer: database.BridgeTransfer{
						CrossDomainMessageHash: &cdh,
						Tx: database.Transaction{
							FromAddress: common.HexToAddress("0x3"),
							ToAddress:   common.HexToAddress("0x4"),
							Timestamp:   5,
						},
						TokenPair: database.TokenPair{
							LocalTokenAddress:  common.HexToAddress("0x6"),
							RemoteTokenAddress: common.HexToAddress("0x7"),
						},
					},
				},
			},
		},
	}

	response := svc.WithdrawResponse(withdraws)
	require.NotEmpty(t, response.Items)
	require.Len(t, response.Items, 1)
	assertFieldsAreSet(t, response.Items[0])
}

func TestDepositResponse(t *testing.T) {
	cdh := common.HexToHash("0x2")
	svc := service.New(nil, nil, nil)

	deposits := &database.L1BridgeDepositsResponse{
		Deposits: []database.L1BridgeDepositWithTransactionHashes{
			{
				L1BridgeDeposit: database.L1BridgeDeposit{
					BridgeTransfer: database.BridgeTransfer{
						CrossDomainMessageHash: &cdh,
						Tx: database.Transaction{
							FromAddress: common.HexToAddress("0x3"),
							ToAddress:   common.HexToAddress("0x4"),
							Timestamp:   5,
						},
						TokenPair: database.TokenPair{
							LocalTokenAddress:  common.HexToAddress("0x6"),
							RemoteTokenAddress: common.HexToAddress("0x7"),
						},
					},
				},
			},
		},
	}

	response := svc.DepositResponse(deposits)
	require.NotEmpty(t, response.Items)
	require.Len(t, response.Items, 1)
	assertFieldsAreSet(t, response.Items[0])
}

func TestQueryParams(t *testing.T) {

	var tests = []struct {
		name string
		test func(*testing.T, service.Service)
	}{
		{
			name: "empty params",
			test: func(t *testing.T, svc service.Service) {
				params, err := svc.QueryParams("", "", "")
				require.Error(t, err)
				require.Nil(t, params)
			},
		},
		{
			name: "empty params except address",
			test: func(t *testing.T, svc service.Service) {
				addr := common.HexToAddress("0x420")
				params, err := svc.QueryParams(addr.String(), "", "")
				require.NoError(t, err)
				require.NotNil(t, params)
				require.Equal(t, addr, params.Address)
				require.Equal(t, 100, params.Limit)
				require.Equal(t, "", params.Cursor)
			},
		},
	}

	v := new(service.Validator)
	svc := service.New(v, nil, log.New())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t, svc)
		})
	}
}
