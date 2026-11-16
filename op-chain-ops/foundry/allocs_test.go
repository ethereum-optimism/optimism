package foundry

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestForgeAllocs_Marshaling(t *testing.T) {
	data := &ForgeAllocs{
		Accounts: map[common.Address]types.Account{
			common.HexToAddress("0x12345"): {
				Balance: big.NewInt(12345),
				Code:    []byte{0x01, 0x02, 0x03},
				Nonce:   123,
				Storage: map[common.Hash]common.Hash{
					common.HexToHash("0x12345"): common.HexToHash("0x12345"),
				},
			},
		},
	}

	out, err := json.Marshal(data)
	require.NoError(t, err)

	var in ForgeAllocs
	require.NoError(t, json.Unmarshal(out, &in))
}
