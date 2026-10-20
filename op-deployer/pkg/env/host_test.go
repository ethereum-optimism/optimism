package env

import (
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestToBroadcast(t *testing.T) {
	for _, typ := range []script.BroadcastType{script.BroadcastCall, script.BroadcastCreate, script.BroadcastCreate2} {
		t.Run(typ.String(), func(t *testing.T) {
			in := script.Broadcast{
				From:    common.Address{0x01},
				To:      common.Address{0x02},
				Input:   hexutil.Bytes{0x03, 0x04},
				Value:   (*hexutil.U256)(uint256.NewInt(5)),
				Salt:    common.Hash{0x06},
				GasUsed: 7,
				Type:    typ,
				Nonce:   8,
			}
			out := toBroadcast(in)

			wantJSON, err := json.Marshal(in)
			require.NoError(t, err)
			gotJSON, err := json.Marshal(out)
			require.NoError(t, err)
			require.JSONEq(t, string(wantJSON), string(gotJSON))
			require.Equal(t, in.ID(), out.ID())
		})
	}
}
