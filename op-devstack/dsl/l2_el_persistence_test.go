package dsl

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDecodePersistedBlock(t *testing.T) {
	persisted := eth.BlockID{Hash: common.HexToHash("0xab"), Number: 42}
	for _, tc := range []struct {
		name         string
		notification string
		want         eth.BlockID
		wantErr      string
	}{
		{name: "block", notification: `{"number":42,"hash":"` + persisted.Hash.Hex() + `"}`, want: persisted},
		{name: "null", notification: `null`, wantErr: "carries no block"},
		{name: "malformed", notification: `{"number":`, wantErr: "undecodable persisted block"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodePersistedBlock([]byte(tc.notification))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
