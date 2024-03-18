package resolved

import (
	"context"
	"math/big"
	"testing"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testClaimants = []common.Address{
	common.Address{0xaa},
}

func TestClaimer_Validate(t *testing.T) {
	t.Run("GameValidationSucceeds", func(t *testing.T) {
		gameAddr := common.HexToAddress("0x1234")
		validator, metrics, contract := newTestClaimValidator(t)
		ctx := context.Background()
		games := []types.GameMetadata{{Proxy: gameAddr}, {Proxy: gameAddr}, {Proxy: gameAddr}}
		err := validator.Validate(ctx, uint64(0), games)
		require.NoError(t, err)
		require.Equal(t, 3, contract.getAllClaimsCalls)
		require.Equal(t, 3, contract.getClaimedBondFlagCalls)
		require.Equal(t, 6, metrics.calls)
	})

	t.Run("ContractCreationFails", func(t *testing.T) {
		validator, _, _ := newTestClaimValidator(t)
		validator.creator = func(game types.GameMetadata) (GameContract, error) {
			return nil, assert.AnError
		}
		err := validator.Validate(context.Background(), uint64(0), []types.GameMetadata{{}})
		require.Error(t, err)
	})

	t.Run("GetAllClaimsFails", func(t *testing.T) {
		validator, _, contract := newTestClaimValidator(t)
		contract.getAllClaimsErr = assert.AnError
		err := validator.Validate(context.Background(), uint64(0), []types.GameMetadata{{}})
		require.Error(t, err)
	})

	t.Run("GetClaimedBondFlagFails", func(t *testing.T) {
		validator, _, contract := newTestClaimValidator(t)
		contract.getClaimedBondFlagErr = assert.AnError
		err := validator.Validate(context.Background(), uint64(0), []types.GameMetadata{{}})
		require.Error(t, err)
	})
}

func newTestClaimValidator(t *testing.T) (*claimValidator, *stubValidatorMetrics, *stubGameContract) {
	logger := testlog.Logger(t, log.LvlDebug)
	m := &stubValidatorMetrics{}
	c := &stubGameContract{}
	creator := func(game types.GameMetadata) (GameContract, error) {
		return c, nil
	}
	return NewClaimValidator(logger, m, creator, testClaimants...), m, c
}

type stubValidatorMetrics struct {
	calls int
}

func (s *stubValidatorMetrics) RecordUnexpectedClaimResolution() {
	s.calls++
}

type stubGameContract struct {
	getAllClaimsCalls       int
	getAllClaimsErr         error
	getClaimedBondFlagCalls int
	getClaimedBondFlagErr   error
}

func newClaim(claimant common.Address, counter common.Address, bond *big.Int) faultTypes.Claim {
	return faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Bond: bond,
		},
		Claimant:    claimant,
		CounteredBy: counter,
	}
}

func (s *stubGameContract) GetAllClaims(ctx context.Context, block rpcblock.Block) ([]faultTypes.Claim, error) {
	s.getAllClaimsCalls++
	if s.getAllClaimsErr != nil {
		return nil, s.getAllClaimsErr
	}
	return []faultTypes.Claim{
		newClaim(testClaimants[0], common.Address{0xbb}, big.NewInt(100)),
		newClaim(testClaimants[0], common.Address{0xbb}, big.NewInt(10)),
		newClaim(testClaimants[0], common.Address{}, big.NewInt(10)),
		newClaim(common.Address{}, common.Address{}, big.NewInt(10)),
		newClaim(testClaimants[0], common.Address{0xbb}, big.NewInt(100)),
		newClaim(testClaimants[0], common.Address{0xbb}, big.NewInt(10)),
		newClaim(testClaimants[0], common.Address{}, big.NewInt(10)),
		newClaim(common.Address{}, common.Address{}, big.NewInt(10)),
	}, nil
}

func (s *stubGameContract) GetClaimedBondFlag(ctx context.Context) (*big.Int, error) {
	s.getClaimedBondFlagCalls++
	if s.getClaimedBondFlagErr != nil {
		return nil, s.getClaimedBondFlagErr
	}
	return big.NewInt(100), nil
}
