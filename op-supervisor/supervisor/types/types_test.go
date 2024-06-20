package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func FuzzRoundtripIdentifierJSONMarshal(f *testing.F) {
	f.Fuzz(func(t *testing.T, origin []byte, blockNumber uint64, logIndex uint64, timestamp uint64, chainID []byte) {
		if len(chainID) > 32 {
			chainID = chainID[:32]
		}

		id := Identifier{
			Origin:      common.BytesToAddress(origin),
			BlockNumber: blockNumber,
			LogIndex:    logIndex,
			Timestamp:   timestamp,
			ChainID:     uint256.Int{},
		}
		id.ChainID.SetBytes(chainID)

		raw, err := json.Marshal(&id)
		require.NoError(t, err)

		var dec Identifier
		require.NoError(t, json.Unmarshal(raw, &dec))

		require.Equal(t, id.Origin, dec.Origin)
		require.Equal(t, id.BlockNumber, dec.BlockNumber)
		require.Equal(t, id.LogIndex, dec.LogIndex)
		require.Equal(t, id.Timestamp, dec.Timestamp)
		require.Equal(t, id.ChainID, dec.ChainID)
	})
}
