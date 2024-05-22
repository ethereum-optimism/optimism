package utils

import (
	"fmt"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

var localNonce uint32 = 0

func GetAccountNonce(accountNonce uint32) uint32 {

	if accountNonce > localNonce {
		localNonce = accountNonce
		return accountNonce
	}
	localNonce++
	return localNonce
}

func GetSubstrateApi(ApiURL string) (*gsrpc.SubstrateAPI, error) {
	api, err := gsrpc.NewSubstrateAPI(ApiURL)

	if err != nil {
		return &gsrpc.SubstrateAPI{}, err
	}
	return api, nil
}

func EnsureValidAppID(appID int) int {
	if appID > 0 {
		return appID
	}
	return 0
}

func GetMetadataLatest(api *gsrpc.SubstrateAPI) (*gsrpc_types.Metadata, error) {

	meta, err := api.RPC.State.GetMetadataLatest()

	if err != nil {
		fmt.Printf("cannot get metadata: error:%v", err)
		return &gsrpc_types.Metadata{}, err
	}

	return meta, err
}

func FetchChainData(api *gsrpc.SubstrateAPI) (gsrpc_types.Hash, *gsrpc_types.RuntimeVersion, error) {
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
