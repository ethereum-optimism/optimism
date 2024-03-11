package utils

import (
	"context"
	"errors"
	"fmt"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum-optimism/optimism/op-plasma/avail/types"
)

func SubmitDataAndWatch(ctx context.Context, data []byte) (types.AvailBlockRef, error) {
	config := GetConfig()
	ApiURL := config.ApiURL
	Seed := config.Seed
	AppID := config.AppID
	api, err := getSubstrateApi(ApiURL)
	if err != nil {
		return types.AvailBlockRef{}, err
	}
	return SubmitAndWait(ctx, api, data, Seed, AppID)
}

func SubmitAndWait(ctx context.Context, api *gsrpc.SubstrateAPI, data []byte, Seed string, AppId int) (types.AvailBlockRef, error) {

	meta, err := getMetadataLatest(api)

	if err != nil {
		return types.AvailBlockRef{}, fmt.Errorf("cannot get substrate API and meta %v", err)
	}
	appID := ensureValidAppID(AppId)

	call, err := createDataAvailabilityCall(meta, data, appID)
	if err != nil {

		return types.AvailBlockRef{}, fmt.Errorf("creating data availability call: %w", err)
	}

	signedExt, sender, nonce, err := prepareAndSignExtrinsic(api, call, meta, Seed, appID)
	if err != nil {
		fmt.Println(err)
		return types.AvailBlockRef{}, fmt.Errorf("preparing and signing extrinsic: %w", err)
	}

	return waitForExtrinsicFinalization(api, signedExt, sender, nonce)

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

	nonce := GetAccountNonce(uint32(accountInfo.Nonce))
	ext := gsrpc_types.NewExtrinsic(call)

	err = signExtrinsic(&ext, keyringPair, genesisHash, rv, nonce, appID)
	if err != nil {
		return gsrpc_types.Extrinsic{}, keyringPair.Address, nonce, err
	}

	return ext, keyringPair.Address, nonce, nil
}

func fetchChainData(api *gsrpc.SubstrateAPI) (gsrpc_types.Hash, *gsrpc_types.RuntimeVersion, error) {
	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return genesisHash, nil, fmt.Errorf("cannot get block hash: %w", err)
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return genesisHash, nil, fmt.Errorf("cannot get runtime version: %w", err)
	}

	return genesisHash, rv, nil
}

func fetchAccountInfo(api *gsrpc.SubstrateAPI, meta *gsrpc_types.Metadata, keyringPair signature.KeyringPair) (gsrpc_types.StorageKey, gsrpc_types.AccountInfo, error) {
	storageKey, err := gsrpc_types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	var accountInfo gsrpc_types.AccountInfo

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

func getMetadataLatest(api *gsrpc.SubstrateAPI) (*gsrpc_types.Metadata, error) {

	meta, err := api.RPC.State.GetMetadataLatest()

	if err != nil {
		fmt.Printf("cannot get metadata: error:%v", err)
		return &gsrpc_types.Metadata{}, err
	}

	return meta, err
}

func waitForExtrinsicFinalization(api *gsrpc.SubstrateAPI, ext gsrpc_types.Extrinsic, sender string, nonce uint32) (types.AvailBlockRef, error) {

	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return types.AvailBlockRef{}, fmt.Errorf("cannot submit extrinsic: %w", err)
	}
	defer sub.Unsubscribe()

	defer sub.Unsubscribe()
	timeout := time.After(100 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsFinalized {
				return types.AvailBlockRef{BlockHash: string(status.AsFinalized.Hex()), Sender: sender, Nonce: int64(nonce)}, nil
			}
		case <-timeout:
			return types.AvailBlockRef{}, errors.New("timitout before getting finalized status")
		}
	}
}
