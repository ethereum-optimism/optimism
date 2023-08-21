package contracts

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	mockStatusError       = errors.New("mock error")
	mockClaimDataError    = fmt.Errorf("claim data errored")
	mockClaimLenError     = fmt.Errorf("claim len errored")
	mockMaxGameDepthError = fmt.Errorf("max game depth errored")
	mockPrestateError     = fmt.Errorf("prestate errored")
)

func TestFaultCaller_GetGameStatus(t *testing.T) {
	t.Run("Succeeds", func(t *testing.T) {
		caller := newMockCaller()
		caller.status = 1
		game := NewFaultDisputeGame(caller)
		status, err := game.GetGameStatus(context.Background())
		require.NoError(t, err)
		require.Equal(t, types.GameStatusChallengerWon, status)
	})

	t.Run("Errors", func(t *testing.T) {
		caller := newMockCaller()
		caller.statusError = true
		game := NewFaultDisputeGame(caller)
		_, err := game.GetGameStatus(context.Background())
		require.ErrorIs(t, err, mockStatusError)
	})
}

func TestFaultCaller_GetClaimCount(t *testing.T) {
	t.Run("Succeeds", func(t *testing.T) {
		game := NewFaultDisputeGame(newMockCaller())
		claimDataLen, err := game.GetClaimCount(context.Background())
		require.EqualValues(t, 3, claimDataLen)
		require.NoError(t, err)
	})

	t.Run("Errors", func(t *testing.T) {
		caller := newMockCaller()
		caller.claimDataLenError = true
		game := NewFaultDisputeGame(caller)
		claimDataLen, err := game.GetClaimCount(context.Background())
		require.Zero(t, claimDataLen)
		require.Equal(t, mockClaimLenError, err)
	})
}

// TestLoader_FetchGameDepth tests [loader.FetchGameDepth].
func TestLoader_FetchGameDepth(t *testing.T) {
	t.Run("Succeeds", func(t *testing.T) {
		mockCaller := newMockCaller()
		mockCaller.maxGameDepth = 10
		game := NewFaultDisputeGame(mockCaller)
		depth, err := game.FetchGameDepth(context.Background())
		require.NoError(t, err)
		require.Equal(t, uint64(10), depth)
	})

	t.Run("Errors", func(t *testing.T) {
		mockCaller := newMockCaller()
		mockCaller.maxGameDepthError = true
		game := NewFaultDisputeGame(mockCaller)
		depth, err := game.FetchGameDepth(context.Background())
		require.ErrorIs(t, mockMaxGameDepthError, err)
		require.Equal(t, depth, uint64(0))
	})
}

// TestLoader_FetchAbsolutePrestateHash tests the [loader.FetchAbsolutePrestateHash] function.
func TestLoader_FetchAbsolutePrestateHash(t *testing.T) {
	t.Run("Succeeds", func(t *testing.T) {
		mockCaller := newMockCaller()
		game := NewFaultDisputeGame(mockCaller)
		prestate, err := game.FetchAbsolutePrestateHash(context.Background())
		require.NoError(t, err)
		require.ElementsMatch(t, common.HexToHash("0xdEad"), prestate)
	})

	t.Run("Errors", func(t *testing.T) {
		mockCaller := newMockCaller()
		mockCaller.prestateError = true
		game := NewFaultDisputeGame(mockCaller)
		prestate, err := game.FetchAbsolutePrestateHash(context.Background())
		require.Error(t, err)
		require.ElementsMatch(t, common.Hash{}, prestate)
	})
}

// TestLoader_FetchClaims_Succeeds tests [loader.FetchClaims].
func TestLoader_FetchClaims_Succeeds(t *testing.T) {
	mockCaller := newMockCaller()
	expectedClaims := mockCaller.returnClaims
	game := NewFaultDisputeGame(mockCaller)
	claims, err := game.FetchClaims(context.Background())
	require.NoError(t, err)
	require.ElementsMatch(t, []types.Claim{
		{
			ClaimData: types.ClaimData{
				Value:    expectedClaims[0].Claim,
				Position: types.NewPositionFromGIndex(expectedClaims[0].Position.Uint64()),
			},
			Parent: types.ClaimData{
				Value:    expectedClaims[0].Claim,
				Position: types.NewPositionFromGIndex(expectedClaims[0].Position.Uint64()),
			},
			Countered:     false,
			Clock:         uint64(0),
			ContractIndex: 0,
		},
		{
			ClaimData: types.ClaimData{
				Value:    expectedClaims[1].Claim,
				Position: types.NewPositionFromGIndex(expectedClaims[1].Position.Uint64()),
			},
			Parent: types.ClaimData{
				Value:    expectedClaims[0].Claim,
				Position: types.NewPositionFromGIndex(expectedClaims[1].Position.Uint64()),
			},
			Countered:     false,
			Clock:         uint64(0),
			ContractIndex: 1,
		},
		{
			ClaimData: types.ClaimData{
				Value:    expectedClaims[2].Claim,
				Position: types.NewPositionFromGIndex(expectedClaims[2].Position.Uint64()),
			},
			Parent: types.ClaimData{
				Value:    expectedClaims[0].Claim,
				Position: types.NewPositionFromGIndex(expectedClaims[2].Position.Uint64()),
			},
			Countered:     false,
			Clock:         uint64(0),
			ContractIndex: 2,
		},
	}, claims)
}

// TestLoader_FetchClaims_ClaimDataErrors tests [loader.FetchClaims]
// when the claim fetcher [ClaimData] function call errors.
func TestLoader_FetchClaims_ClaimDataErrors(t *testing.T) {
	mockCaller := newMockCaller()
	mockCaller.claimDataError = true
	game := NewFaultDisputeGame(mockCaller)
	claims, err := game.FetchClaims(context.Background())
	require.ErrorIs(t, err, mockClaimDataError)
	require.Empty(t, claims)
}

// TestLoader_FetchClaims_ClaimLenErrors tests [loader.FetchClaims]
// when the claim fetcher [ClaimDataLen] function call errors.
func TestLoader_FetchClaims_ClaimLenErrors(t *testing.T) {
	mockCaller := newMockCaller()
	mockCaller.claimDataLenError = true
	game := NewFaultDisputeGame(mockCaller)
	claims, err := game.FetchClaims(context.Background())
	require.ErrorIs(t, err, mockClaimLenError)
	require.Empty(t, claims)
}

type mockFaultDisputeGameCaller struct {
	statusError       bool
	claimDataLenError bool
	claimDataError    bool
	maxGameDepthError bool
	prestateError     bool

	status       uint8
	maxGameDepth uint64
	currentIndex uint64
	returnClaims []struct {
		ParentIndex uint32
		Countered   bool
		Claim       [32]byte
		Position    *big.Int
		Clock       *big.Int
	}
}

func newMockCaller() *mockFaultDisputeGameCaller {
	return &mockFaultDisputeGameCaller{
		returnClaims: []struct {
			ParentIndex uint32
			Countered   bool
			Claim       [32]byte
			Position    *big.Int
			Clock       *big.Int
		}{
			{
				Claim:     [32]byte{0x00},
				Position:  big.NewInt(0),
				Countered: false,
				Clock:     big.NewInt(0),
			},
			{
				Claim:     [32]byte{0x01},
				Position:  big.NewInt(0),
				Countered: false,
				Clock:     big.NewInt(0),
			},
			{
				Claim:     [32]byte{0x02},
				Position:  big.NewInt(0),
				Countered: false,
				Clock:     big.NewInt(0),
			},
		},
	}
}

func (m *mockFaultDisputeGameCaller) Status(opts *bind.CallOpts) (uint8, error) {
	if m.statusError {
		return 0, mockStatusError
	}
	return m.status, nil
}

func (m *mockFaultDisputeGameCaller) ClaimDataLen(opts *bind.CallOpts) (*big.Int, error) {
	if m.claimDataLenError {
		return nil, mockClaimLenError
	}
	return big.NewInt(int64(len(m.returnClaims))), nil
}

func (m *mockFaultDisputeGameCaller) ClaimData(opts *bind.CallOpts, arg0 *big.Int) (struct {
	ParentIndex uint32
	Countered   bool
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}, error) {
	if m.claimDataError {
		return struct {
			ParentIndex uint32
			Countered   bool
			Claim       [32]byte
			Position    *big.Int
			Clock       *big.Int
		}{}, mockClaimDataError
	}
	returnClaim := m.returnClaims[m.currentIndex]
	m.currentIndex++
	return returnClaim, nil
}

func (m *mockFaultDisputeGameCaller) MAXGAMEDEPTH(opts *bind.CallOpts) (*big.Int, error) {
	if m.maxGameDepthError {
		return nil, mockMaxGameDepthError
	}
	return big.NewInt(int64(m.maxGameDepth)), nil
}
func (m *mockFaultDisputeGameCaller) ABSOLUTEPRESTATE(opts *bind.CallOpts) ([32]byte, error) {
	if m.prestateError {
		return [32]byte{}, mockPrestateError
	}
	return common.HexToHash("0xdEad"), nil
}
