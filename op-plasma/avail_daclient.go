package plasma

import (
	"context"
	"errors"
	"fmt"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum-optimism/optimism/op-plasma/avail/config"
	"github.com/ethereum-optimism/optimism/op-plasma/avail/types"
	"github.com/ethereum-optimism/optimism/op-plasma/avail/utils"
)

const cfgPath = "../"

type AvailDAClient struct {
	*DAClient
}

// GetInput returns the input data corresponding to a given commitment bytes
func (c *AvailDAClient) GetInput(ctx context.Context, key []byte) ([]byte, error) {
	return nil, nil
}

// SetInput sets the input data and returns the KZG commitment hash from Avail DA
func (c *AvailDAClient) SetInput(ctx context.Context, img []byte) ([]byte, error) {
	if len(img) >= 512000 {
		return []byte{}, fmt.Errorf("size of TxData is more than 512KB, it is higher than a single data submit transaction supports on avail")
	}

	avail_Blk_Ref, err := SubmitDataAndWatch(ctx, img)

	if err != nil {
		return []byte{}, fmt.Errorf("cannot submit data:%v", err)
	}

	ref_bytes_data, err := avail_Blk_Ref.MarshalToBinary()

	if err != nil {
		return []byte{}, fmt.Errorf("cannot get the binary form of avail block reference:%v", err)
	}
	return ref_bytes_data, nil
}

// submitData creates a transaction and makes a Avail data submission
func SubmitDataAndWatch(ctx context.Context, data []byte) (types.AvailBlockRef, error) {
	//Load variables
	var config config.Config
	err := config.GetConfig(cfgPath)
	if err != nil {
		panic(fmt.Sprintf("cannot get config:%v", err))
	}

	ApiURL := config.ApiURL
	Seed := config.Seed
	AppID := config.AppID

	api, meta, err := getSubstrateApiAndMeta(ApiURL)
	if err != nil {
		fmt.Printf("cannot get substrate API and meta %v", err)
	}

	appID := ensureValidAppID(AppID)

	call, err := createDataAvailabilityCall(meta, data, appID)
	if err != nil {
		return types.AvailBlockRef{}, fmt.Errorf("creating data availability call: %w", err)
	}

	signedExt, sender, nonce, err := prepareAndSignExtrinsic(api, call, meta, Seed, appID)
	if err != nil {
		return types.AvailBlockRef{}, fmt.Errorf("preparing and signing extrinsic: %w", err)
	}

	return waitForExtrinsicFinalization(ctx, api, signedExt, sender, nonce)
}
func prepareAndSignExtrinsic(api *gsrpc.SubstrateAPI, call gsrpc_types.Call, meta *gsrpc_types.Metadata, seed string, appID int) (gsrpc_types.Extrinsic, string, uint32, error) {
	genesisHash, rv, err := fetchChainData(api)
	if err != nil {
		return gsrpc_types.Extrinsic{}, "", 0, err
	}

	keyringPair, err := signature.KeyringPairFromSecret(seed, 42)
	if err != nil {
		return gsrpc_types.Extrinsic{}, "", 0, fmt.Errorf("cannot create key pair: %w", err)
	}

	_, accountInfo, err := fetchAccountInfo(api, meta, keyringPair)
	if err != nil {
		return gsrpc_types.Extrinsic{}, keyringPair.Address, 0, err
	}

	nonce := utils.GetAccountNonce(uint32(accountInfo.Nonce))
	ext := gsrpc_types.NewExtrinsic(call)

	err = signExtrinsic(&ext, keyringPair, genesisHash, rv, nonce, appID)
	if err != nil {
		return gsrpc_types.Extrinsic{}, keyringPair.Address, nonce, err
	}

	return ext, keyringPair.Address, nonce, nil
}

func fetchChainData(api *gsrpc.SubstrateAPI) (genesisHash gsrpc_types.Hash, rv *gsrpc_types.RuntimeVersion, err error) {
	genesisHash, err = api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return genesisHash, nil, fmt.Errorf("cannot get block hash: %w", err)
	}

	rv, err = api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return genesisHash, nil, fmt.Errorf("cannot get runtime version: %w", err)
	}

	return genesisHash, rv, nil
}

func fetchAccountInfo(api *gsrpc.SubstrateAPI, meta *gsrpc_types.Metadata, keyringPair signature.KeyringPair) (storageKey gsrpc_types.StorageKey, accountInfo gsrpc_types.AccountInfo, err error) {
	storageKey, err = gsrpc_types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		return storageKey, accountInfo, fmt.Errorf("cannot create storage key: %w", err)
	}

	ok, err := api.RPC.State.GetStorageLatest(storageKey, &accountInfo)
	if err != nil || !ok {
		return storageKey, accountInfo, fmt.Errorf("cannot get latest storage: %w", err)
	}

	return storageKey, accountInfo, nil
}

func signExtrinsic(ext *gsrpc_types.Extrinsic, keyringPair signature.KeyringPair, genesisHash gsrpc_types.Hash, rv *gsrpc_types.RuntimeVersion, nonce uint32, appID int) error {
	options := gsrpc_types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                gsrpc_types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              gsrpc_types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                gsrpc_types.NewUCompactFromUInt(0),
		AppID:              gsrpc_types.NewUCompactFromUInt(uint64(appID)),
		TransactionVersion: rv.TransactionVersion,
	}

	err := ext.Sign(keyringPair, options)
	if err != nil {
		return fmt.Errorf("cannot sign extrinsic: %w", err)
	}

	return nil
}

func createDataAvailabilityCall(meta *gsrpc_types.Metadata, data []byte, appID int) (gsrpc_types.Call, error) {
	c, err := gsrpc_types.NewCall(meta, "DataAvailability.submit_data", gsrpc_types.NewBytes(data))
	if err != nil {
		return gsrpc_types.Call{}, fmt.Errorf("cannot create new call: %w", err)
	}
	return c, nil
}

func ensureValidAppID(appID int) int {
	if appID > 0 {
		return appID
	}
	return 0
}

func getSubstrateApiAndMeta(ApiURL string) (*gsrpc.SubstrateAPI, *gsrpc_types.Metadata, error) {
	api, err := gsrpc.NewSubstrateAPI(ApiURL)
	if err != nil {
		fmt.Printf("cannot create api: error:%v", err)
		return &gsrpc.SubstrateAPI{}, &gsrpc_types.Metadata{}, err
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		fmt.Printf("cannot get metadata: error:%v", err)
		return &gsrpc.SubstrateAPI{}, &gsrpc_types.Metadata{}, err
	}

	return api, meta, err
}

func waitForExtrinsicFinalization(ctx context.Context, api *gsrpc.SubstrateAPI, ext gsrpc_types.Extrinsic, sender string, nonce uint32) (types.AvailBlockRef, error) {
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return types.AvailBlockRef{}, fmt.Errorf("cannot submit extrinsic: %w", err)
	}
	defer sub.Unsubscribe()

	select {
	case status := <-sub.Chan():
		if status.IsFinalized {
			return types.AvailBlockRef{BlockHash: string(status.AsFinalized.Hex()), Sender: sender, Nonce: int64(nonce)}, nil
		}
	case <-ctx.Done():
		return types.AvailBlockRef{}, ctx.Err()
	case <-time.After(100 * time.Second):
		return types.AvailBlockRef{}, errors.New("timeout before getting finalized status")
	}
	return types.AvailBlockRef{}, errors.New("unexpected error waiting for extrinsic finalization")
}
