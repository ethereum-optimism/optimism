package models_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ethereum-optimism/optimism/indexer/api/models"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCreateWithdrawal(t *testing.T) {
	// (1) Create a dummy database response object

	cdh := common.HexToHash("0x2")
	dbWithdrawals := &database.L2BridgeWithdrawalsResponse{
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

	// (2) Create and validate response object

	response := models.CreateWithdrawalResponse(dbWithdrawals)
	require.NotEmpty(t, response.Items)
	require.Len(t, response.Items, 1)

	// (3) Use reflection to check that all fields in WithdrawalItem are populated correctly

	item := response.Items[0]
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
