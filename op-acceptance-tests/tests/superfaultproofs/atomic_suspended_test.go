package superfaultproofs

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestAtomicPlanPreservesReplayedEnvelope(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signed, err := types.SignNewTx(key, types.LatestSignerForChainID(big.NewInt(901)), &types.DynamicFeeTx{ChainID: big.NewInt(901), Nonce: 7, Gas: 100_000, GasFeeCap: big.NewInt(99), GasTipCap: big.NewInt(3)})
	require.NoError(t, err)
	planned := fixedAtomicPlan(signed)
	txplan.WithStaticNonce(99)(planned)
	txplan.WithGasLimit(200_000)(planned)
	got, err := planned.Signed.Eval(t.Context())
	require.NoError(t, err)
	expected, err := signed.MarshalBinary()
	require.NoError(t, err)
	actual, err := got.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
