package mocks

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type AvailMockRPC struct {
}

func (rpc *AvailMockRPC) GetMetadataLatest() *types.Metadata {
	return types.NewMetadataV10()
}

func (rpc *AvailMockRPC) GetRuntimeVersionLatest() *types.RuntimeVersion {
	return types.NewRuntimeVersion()
}

func (rpc *AvailMockRPC) GetStorageLatest(key types.StorageKey, target interface{}) (ok bool, err error) {
	return true, err
}
