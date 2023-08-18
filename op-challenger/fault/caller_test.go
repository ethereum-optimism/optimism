package fault

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

var (
	errMock = errors.New("mock error")
)

type mockFaultDisputeGameCaller struct {
	status    uint8
	errStatus bool

	claimDataLen    *big.Int
	errClaimDataLen bool
}

func (m *mockFaultDisputeGameCaller) Status(opts *bind.CallOpts) (uint8, error) {
	if m.errStatus {
		return 0, errMock
	}
	return m.status, nil
}

func (m *mockFaultDisputeGameCaller) ClaimDataLen(opts *bind.CallOpts) (*big.Int, error) {
	if m.errClaimDataLen {
		return nil, errMock
	}
	return m.claimDataLen, nil
}

func TestFaultCaller_GetGameStatus(t *testing.T) {
	tests := []struct {
		name           string
		caller         FaultDisputeGameCaller
		expectedStatus types.GameStatus
		expectedErr    error
	}{
		{
			name: "success",
			caller: &mockFaultDisputeGameCaller{
				status: 1,
			},
			expectedStatus: types.GameStatusChallengerWon,
			expectedErr:    nil,
		},
		{
			name: "error",
			caller: &mockFaultDisputeGameCaller{
				errStatus: true,
			},
			expectedStatus: 0,
			expectedErr:    errMock,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fc := NewFaultCaller(test.caller, nil)
			status, err := fc.GetGameStatus(context.Background())
			require.Equal(t, test.expectedStatus, status)
			require.Equal(t, test.expectedErr, err)
		})
	}
}

func TestFaultCaller_GetClaimDataLength(t *testing.T) {
	tests := []struct {
		name                 string
		caller               FaultDisputeGameCaller
		expectedClaimDataLen *big.Int
		expectedErr          error
	}{
		{
			name: "success",
			caller: &mockFaultDisputeGameCaller{
				claimDataLen: big.NewInt(1),
			},
			expectedClaimDataLen: big.NewInt(1),
			expectedErr:          nil,
		},
		{
			name: "error",
			caller: &mockFaultDisputeGameCaller{
				errClaimDataLen: true,
			},
			expectedClaimDataLen: nil,
			expectedErr:          errMock,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fc := NewFaultCaller(test.caller, nil)
			claimDataLen, err := fc.GetClaimDataLength(context.Background())
			require.Equal(t, test.expectedClaimDataLen, claimDataLen)
			require.Equal(t, test.expectedErr, err)
		})
	}
}
