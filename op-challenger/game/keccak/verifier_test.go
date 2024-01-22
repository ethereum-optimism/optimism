package keccak

import (
	"bytes"
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/fetcher"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestValidLeaves(t *testing.T) {
	dataLen := uint32(500)
	rng := rand.New(rand.NewSource(842984))
	data := testutils.RandomData(rng, int(dataLen))
	preimage := types.LargePreimageMetaData{
		LargePreimageIdent: types.LargePreimageIdent{},
		Timestamp:          100,
		PartOffset:         0,
		ClaimedSize:        dataLen,
		BlocksProcessed:    2,
		BytesProcessed:     dataLen,
		Countered:          false,
	}
	verifier, fetcher, oracle := setupVerifierTest(t)
	fetcher.leaves = computeValidLeafs(t, data)
	err := verifier.Verify(context.Background(), common.Hash{0xaa}, oracle, preimage)
	require.NoError(t, err)
}

func computeValidLeafs(t *testing.T, data []byte) []types.Leaf {
	matrix := matrix.NewStateMatrix()
	leaves, err := matrix.AbsorbAll(bytes.NewReader(data))
	require.NoError(t, err)
	return leaves
}

func setupVerifierTest(t *testing.T) (*PreimageVerifier, *stubFetcher, *stubOracle) {
	logger := testlog.Logger(t, log.LvlInfo)
	fetcher := &stubFetcher{}
	oracle := &stubOracle{}
	verifier := NewPreimageVerifier(logger, fetcher)
	return verifier, fetcher, oracle
}

type stubFetcher struct {
	leaves []types.Leaf
}

func (s *stubFetcher) FetchLeaves(_ context.Context, _ common.Hash, oracle fetcher.Oracle, ident types.LargePreimageIdent) ([]types.Leaf, error) {
	return s.leaves, nil
}
