package client

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestBootstrapClient(t *testing.T) {
	bootInfo := &BootInfo{
		L1Head:             common.HexToHash("0x1111"),
		L2Head:             common.HexToHash("0x2222"),
		L2Claim:            common.HexToHash("0x3333"),
		L2ClaimBlockNumber: 1,
		L2ChainConfig:      params.GoerliChainConfig,
		RollupConfig:       &chaincfg.Goerli,
	}
	mockOracle := &mockBoostrapOracle{bootInfo}
	readBootInfo := NewBootstrapClient(mockOracle).BootInfo()
	require.EqualValues(t, bootInfo, readBootInfo)
}

type mockBoostrapOracle struct {
	b *BootInfo
}

func (o *mockBoostrapOracle) Get(key preimage.Key) []byte {
	switch key.PreimageKey() {
	case L1HeadLocalIndex.PreimageKey():
		return o.b.L1Head[:]
	case L2HeadLocalIndex.PreimageKey():
		return o.b.L2Head[:]
	case L2ClaimLocalIndex.PreimageKey():
		return o.b.L2Claim[:]
	case L2ClaimBlockNumberLocalIndex.PreimageKey():
		return binary.BigEndian.AppendUint64(nil, o.b.L2ClaimBlockNumber)
	case L2ChainConfigLocalIndex.PreimageKey():
		b, _ := json.Marshal(o.b.L2ChainConfig)
		return b
	case RollupConfigLocalIndex.PreimageKey():
		b, _ := json.Marshal(o.b.RollupConfig)
		return b
	default:
		panic("unknown key")
	}
}
