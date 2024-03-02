package plasma

import (
	"context"
	"math/rand"
	"testing"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	mocks "github.com/ethereum-optimism/optimism/op-plasma/avail/mocks"
	mockgen "github.com/ethereum-optimism/optimism/op-plasma/avail/mocks/mockgen"
	"github.com/ethereum-optimism/optimism/op-plasma/avail/types"
	utils "github.com/ethereum-optimism/optimism/op-plasma/avail/utils"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func SubmitDataAndWatchMock(ctx context.Context, api *gsrpc.SubstrateAPI, data []byte) (types.AvailBlockRef, error) {
	config := utils.GetConfig()

	ApiURL := config.ApiURL
	Seed := config.Seed
	AppID := config.AppID

	return utils.SubmitAndWait(ctx, api, data, ApiURL, Seed, AppID)
}

func TestAvailDAClient(t *testing.T) {

	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRPC := mockgen.NewMockRPCInterface(ctrl)
	mockAuthor := mockgen.NewMockAvailAuthor(ctrl)
	mockState := mockgen.NewMockAvailState(ctrl)
	mockChain := mockgen.NewMockAvailChain(ctrl)

	cfg := CLIConfig{
		Enabled:      true,
		DAServerURL:  "wss://goldberg.avail.tools/ws",
		VerifyOnRead: true,
		UseAvailDA:   true,
	}
	require.NoError(t, cfg.Check())

	mockImplementation := mocks.AvailMockRPC{}

	rng := rand.New(rand.NewSource(1234))
	input := testutils.RandomData(rng, 2000)

	var accountInfo gsrpc_types.AccountInfo

	mockRPC.EXPECT().Author().Return(mockAuthor)
	mockRPC.EXPECT().Chain().Return(mockChain)
	mockRPC.EXPECT().State().Return(mockState)

	mockState.EXPECT().GetMetadataLatest().Return(mockImplementation.GetMetadataLatest()).Times(1)
	mockChain.EXPECT().GetBlockHash(0).Return(mockImplementation.GetBlockHash(0)).Times(1)
	mockState.EXPECT().GetRuntimeVersionLatest().Return(mockImplementation.GetRuntimeVersionLatest()).Times(1)
	mockState.EXPECT().GetStorageLatest(gsrpc_types.StorageKey{}, &accountInfo).Return(mockImplementation.GetStorageLatest(gsrpc_types.StorageKey{}, &accountInfo)).Times(1)

	// SubmitDataAndWatchMock(ctx, &gsrpc.SubstrateAPI{}, input)
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
