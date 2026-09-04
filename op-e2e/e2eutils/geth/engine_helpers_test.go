package geth

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	opeth "github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestEnsureAmsterdamBlockAccessList(t *testing.T) {
	payload := new(engine.ExecutableData)
	EnsureAmsterdamBlockAccessList(payload)
	require.NotNil(t, payload.BlockAccessList)
	require.Equal(t, hexutil.Bytes(rlp.EmptyList), *payload.BlockAccessList)

	existing := hexutil.Bytes{0xc1, 0x80}
	payload.BlockAccessList = &existing
	EnsureAmsterdamBlockAccessList(payload)
	require.Same(t, &existing, payload.BlockAccessList)
}

func TestValidatePayloadStatus(t *testing.T) {
	require.NoError(t, ValidatePayloadStatus("test", engine.PayloadStatusV1{Status: engine.VALID}))
	require.EqualError(t, ValidatePayloadStatus("test", engine.PayloadStatusV1{Status: engine.SYNCING}), "test returned payload status SYNCING")

	validationError := "invalid block access list"
	require.EqualError(t,
		ValidatePayloadStatus("test", engine.PayloadStatusV1{Status: engine.INVALID, ValidationError: &validationError}),
		"test returned payload status INVALID: invalid block access list",
	)
}

func TestFakePoSAdvancesAmsterdamChain(t *testing.T) {
	activation := uint64(0)
	blobSchedule := &params.BlobScheduleConfig{
		Cancun:    params.DefaultCancunBlobConfig,
		Prague:    params.DefaultPragueBlobConfig,
		Osaka:     params.DefaultOsakaBlobConfig,
		BPO1:      params.DefaultBPO1BlobConfig,
		BPO2:      params.DefaultBPO2BlobConfig,
		BPO3:      params.DefaultBPO3BlobConfig,
		BPO4:      params.DefaultBPO4BlobConfig,
		BPO5:      params.DefaultBPO4BlobConfig,
		Amsterdam: params.DefaultBPO4BlobConfig,
	}
	l1Genesis, err := genesis.NewL1GenesisMinimal(&genesis.DevL1DeployConfigMinimal{
		DevL1DeployConfig: genesis.DevL1DeployConfig{
			L1GenesisBlockTimestamp: hexutil.Uint64(time.Now().Unix()),
		},
		L1ChainID:             opeth.ChainIDFromUInt64(900),
		L1PragueTimeOffset:    &activation,
		L1OsakaTimeOffset:     &activation,
		L1BPO1TimeOffset:      &activation,
		L1BPO2TimeOffset:      &activation,
		L1BPO3TimeOffset:      &activation,
		L1BPO4TimeOffset:      &activation,
		L1BPO5TimeOffset:      &activation,
		L1AmsterdamTimeOffset: &activation,
		BlobScheduleConfig:    blobSchedule,
	})
	require.NoError(t, err)

	logicalClock := clock.NewAdvancingClock()
	instance, fakePoS, err := InitL1(1, 64, l1Genesis, logicalClock, t.TempDir(), noOpBeacon{})
	require.NoError(t, err)
	pausedAtBlock1, err := fakePoS.PauseAtBlock(1)
	require.NoError(t, err)
	require.NoError(t, instance.Node.Start())
	t.Cleanup(func() { require.NoError(t, instance.Close()) })

	logicalClock.AdvanceTime(2 * time.Second)
	select {
	case <-pausedAtBlock1:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for fake PoS to pause at block 1")
	}
	require.Equal(t, big.NewInt(1), instance.Backend.BlockChain().CurrentBlock().Number)

	pausedAtBlock2, err := fakePoS.PauseAtBlock(2)
	require.NoError(t, err)
	logicalClock.AdvanceTime(10 * time.Second)
	require.Equal(t, big.NewInt(1), instance.Backend.BlockChain().CurrentBlock().Number)
	require.NoError(t, fakePoS.Resume())
	select {
	case <-pausedAtBlock2:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for fake PoS to pause at block 2")
	}
	require.Equal(t, big.NewInt(2), instance.Backend.BlockChain().CurrentBlock().Number)
}

type noOpBeacon struct{}

func (noOpBeacon) StoreBlobsBundle(uint64, *engine.BlobsBundle) error {
	return nil
}
