package keccak

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/fetcher"
	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/matrix"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestVerify(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	tests := []struct {
		name        string
		inputs      func() []keccakTypes.InputData
		expectedErr error
	}{
		{
			name:        "Valid-SingleInput",
			inputs:      func() []keccakTypes.InputData { return validInputs(t, 1) },
			expectedErr: matrix.ErrValid,
		},
		{
			name:        "Valid-MultipleInputs",
			inputs:      func() []keccakTypes.InputData { return validInputs(t, 3) },
			expectedErr: matrix.ErrValid,
		},
		{
			name: "Invalid-FirstCommitment",
			inputs: func() []keccakTypes.InputData {
				inputs := validInputs(t, 1)
				inputs[0].Commitments[0] = common.Hash{0xaa}
				return inputs
			},
			expectedErr: nil,
		},
		{
			name: "Invalid-MiddleCommitment",
			inputs: func() []keccakTypes.InputData {
				inputs := validInputs(t, 1)
				inputs[0].Commitments[1] = common.Hash{0xaa}
				return inputs
			},
			expectedErr: nil,
		},
		{
			name: "Invalid-LastCommitment",
			inputs: func() []keccakTypes.InputData {
				inputs := validInputs(t, 3)
				inputs[2].Commitments[len(inputs[2].Commitments)-1] = common.Hash{0xaa}
				return inputs
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			fetcher := &stubFetcher{
				inputs: test.inputs(),
			}
			verifier := NewPreimageVerifier(logger, fetcher)
			preimage := keccakTypes.LargePreimageMetaData{}
			challenge, err := verifier.CreateChallenge(context.Background(), common.Hash{0xff}, &stubOracle{}, preimage)
			require.ErrorIs(t, err, test.expectedErr)
			if err == nil {
				// Leave checking the validity of the challenge to the StateMatrix tests
				// Just confirm that we got a non-zero challenge
				require.NotEqual(t, keccakTypes.Challenge{}, challenge)
			} else {
				require.Equal(t, keccakTypes.Challenge{}, challenge)
			}
		})
	}
}

func validInputs(t *testing.T, inputCount int) []keccakTypes.InputData {
	chunkSize := 2 * keccakTypes.BlockSize
	data := testutils.RandomData(rand.New(rand.NewSource(4444)), inputCount*chunkSize)
	var calls []keccakTypes.InputData
	in := bytes.NewReader(data)
	s := matrix.NewStateMatrix()
	for {
		call, err := s.AbsorbUpTo(in, chunkSize)
		if !errors.Is(err, io.EOF) {
			require.NoError(t, err)
		}
		calls = append(calls, call)
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return calls
}

type stubFetcher struct {
	inputs []keccakTypes.InputData
}

func (s *stubFetcher) FetchInputs(_ context.Context, _ common.Hash, _ fetcher.Oracle, _ keccakTypes.LargePreimageIdent) ([]keccakTypes.InputData, error) {
	return s.inputs, nil
}
