package fault

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	mockClaimDataError    = fmt.Errorf("claim data errored")
	mockClaimLenError     = fmt.Errorf("claim len errored")
	mockMaxGameDepthError = fmt.Errorf("max game depth errored")
	mockPrestateError     = fmt.Errorf("prestate errored")
)

type mockCaller struct {
	claimDataError    bool
	claimLenError     bool
	maxGameDepthError bool
	prestateError     bool
	maxGameDepth      uint64
	currentIndex      uint64
	returnClaims      []struct {
		ParentIndex uint32
		Countered   bool
		Claim       [32]byte
		Position    *big.Int
		Clock       *big.Int
	}
}

func newMockCaller() *mockCaller {
	return &mockCaller{
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

func (m *mockCaller) ClaimData(opts *bind.CallOpts, arg0 *big.Int) (struct {
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

func (m *mockCaller) ClaimDataLen(opts *bind.CallOpts) (*big.Int, error) {
	if m.claimLenError {
		return big.NewInt(0), mockClaimLenError
	}
	return big.NewInt(int64(len(m.returnClaims))), nil
}

func (m *mockCaller) MAXGAMEDEPTH(opts *bind.CallOpts) (*big.Int, error) {
	if m.maxGameDepthError {
		return nil, mockMaxGameDepthError
	}
	return big.NewInt(int64(m.maxGameDepth)), nil
}

func (m *mockCaller) ABSOLUTEPRESTATE(opts *bind.CallOpts) ([32]byte, error) {
	if m.prestateError {
		return [32]byte{}, mockPrestateError
	}
	return common.HexToHash("0xdEad"), nil
}

// TestLoader_FetchGameDepth tests [loader.FetchGameDepth].
func TestLoader_FetchGameDepth(t *testing.T) {
	t.Run("Succeeds", func(t *testing.T) {
		mockCaller := newMockCaller()
		mockCaller.maxGameDepth = 10
		loader := NewLoader(mockCaller)
		depth, err := loader.FetchGameDepth(context.Background())
		require.NoError(t, err)
		require.Equal(t, uint64(10), depth)
	})

	t.Run("Errors", func(t *testing.T) {
		mockCaller := newMockCaller()
		mockCaller.maxGameDepthError = true
		loader := NewLoader(mockCaller)
		depth, err := loader.FetchGameDepth(context.Background())
		require.ErrorIs(t, mockMaxGameDepthError, err)
		require.Equal(t, depth, uint64(0))
	})
}

// TestLoader_FetchAbsolutePrestateHash tests the [loader.FetchAbsolutePrestateHash] function.
func TestLoader_FetchAbsolutePrestateHash(t *testing.T) {
	t.Run("Succeeds", func(t *testing.T) {
		mockCaller := newMockCaller()
		loader := NewLoader(mockCaller)
		prestate, err := loader.FetchAbsolutePrestateHash(context.Background())
		require.NoError(t, err)
		require.ElementsMatch(t, common.HexToHash("0xdEad"), prestate)
	})

	t.Run("Errors", func(t *testing.T) {
		mockCaller := newMockCaller()
		mockCaller.prestateError = true
		loader := NewLoader(mockCaller)
		prestate, err := loader.FetchAbsolutePrestateHash(context.Background())
		require.Error(t, err)
		require.ElementsMatch(t, common.Hash{}, prestate)
	})
}

// TestLoader_FetchClaims_Succeeds tests [loader.FetchClaims].
func TestLoader_FetchClaims_Succeeds(t *testing.T) {
	mockCaller := newMockCaller()
	expectedClaims := mockCaller.returnClaims
	loader := NewLoader(mockCaller)
	claims, err := loader.FetchClaims(context.Background())
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
	loader := NewLoader(mockCaller)
	claims, err := loader.FetchClaims(context.Background())
	require.ErrorIs(t, err, mockClaimDataError)
	require.Empty(t, claims)
}

// TestLoader_FetchClaims_ClaimLenErrors tests [loader.FetchClaims]
// when the claim fetcher [ClaimDataLen] function call errors.
func TestLoader_FetchClaims_ClaimLenErrors(t *testing.T) {
	mockCaller := newMockCaller()
	mockCaller.claimLenError = true
	loader := NewLoader(mockCaller)
	claims, err := loader.FetchClaims(context.Background())
	require.ErrorIs(t, err, mockClaimLenError)
	require.Empty(t, claims)
}
