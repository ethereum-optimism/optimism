package plasma

import (
	"context"
	"math/rand"
	"testing"

	utils "github.com/ethereum-optimism/optimism/op-plasma/avail/utils"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/stretchr/testify/require"
)

func TestAvailDAClient(t *testing.T) {

	//server := mockServer.New()

	ctx := context.Background()

	// ctrl := gomock.NewController(t)
	// defer ctrl.Finish()

	// Author := mocks.NewAuthor(t)
	// Beefy := mocks.NewBeefy(t)
	// Chain := mocks.NewChain(t)
	// MMR := mocks.NewMMR(t)
	// Offchain := mocks.NewOffchain(t)
	// State := mocks.NewState(t)
	// System := mocks.NewSystem(t)

	// rpc := rpc.RPC{
	// 	Author:   Author,
	// 	Beefy:    Beefy,
	// 	Chain:    Chain,
	// 	MMR:      MMR,
	// 	Offchain: Offchain,
	// 	State:    State,
	// 	System:   System,
	// }

	// mockData, err := mocks.GenerateMockMetadata()
	// if err != nil {
	// 	panic(err)
	// }
	// // Author.On("SubmitAndWatchExtrinsic").Return()
	// State.On("GetMetadataLatest").Return(mockData, nil)

	// // gs.AsMetadataV10.Modules[0].

	// State.On("GetRuntimeVersionLatest").Return(gsrpc_types.NewRuntimeVersion(), nil).Times(3)
	// State.On("GetStorageLatest", gsrpc_types.StorageKey{}, &gsrpc_types.AccountInfo{}).Return(func(key gsrpc_types.StorageKey, target *gsrpc_types.AccountInfo) bool {
	// 	target.Nonce = 10
	// 	return true
	// }, nil).Times(3)
	// Chain.On("GetBlockHash", uint64(0)).Return(gsrpc_types.NewHashFromHexString("0xb226886ccc5595edc7a54458183c9c487dc7df8da255455fb97a0dc79588b839"))

	cfg := CLIConfig{
		Enabled:      true,
		DAServerURL:  "wss://goldberg.avail.tools/ws",
		VerifyOnRead: true,
		UseAvailDA:   true,
	}
	require.NoError(t, cfg.Check())

	rng := rand.New(rand.NewSource(1234))
	input := testutils.RandomData(rng, 2000)

	_, err := utils.SubmitDataAndWatch(ctx, input)
	require.NoError(t, err)

	// comm, err := client.SetInput(ctx, input)
	// fmt.Println("comm", comm, err)
	// require.NoError(t, err)

	// require.Equal(t, comm, crypto.Keccak256(input))

	// stored, err := client.GetInput(ctx, comm)
	// require.NoError(t, err)

	// require.Equal(t, input, stored)

	// // set a bad commitment in the store
	// require.NoError(t, store.Put(comm, []byte("bad data")))

	// _, err = client.GetInput(ctx, comm)
	// require.ErrorIs(t, err, ErrCommitmentMismatch)

	// // test not found error
	// comm = crypto.Keccak256(testutils.RandomData(rng, 32))
	// _, err = client.GetInput(ctx, comm)
	// require.ErrorIs(t, err, ErrNotFound)

	// // test storing bad data
	// _, err = client.SetInput(ctx, []byte{})
	// require.ErrorIs(t, err, ErrInvalidInput)

	// // server not responsive
	// tsrv.Close()
	// _, err = client.SetInput(ctx, input)
	// require.Error(t, err)

	// _, err = client.GetInput(ctx, crypto.Keccak256(input))
	// require.Error(t, err)
}
