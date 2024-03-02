package mocks

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type AvailChain interface {
	GetBlockHash(uint64) (types.Hash, error)
}

type AvailState interface {
	GetMetadataLatest() (*types.Metadata, error)
	GetRuntimeVersionLatest() *types.RuntimeVersion
	GetStorageLatest(key gsrpc_types.StorageKey, target interface{}) (ok bool, err error)
}

type AvailAuthor interface {
	SubmitAndWatchExtrinsic(ext gsrpc_types.Extrinsic) (*author.ExtrinsicStatusSubscription, error)
}

type RPCInterface interface {
	Chain() AvailChain
	State() AvailState
	Author() AvailAuthor
}

type AvailRPC interface {
	RPC() RPCInterface
}
type AvailMockRPC struct {
}

func (rpc *AvailMockRPC) GetBlockHash(blockNumber uint64) (types.Hash, error) {
	return types.NewHashFromHexString("0xb226886ccc5595edc7a54458183c9c487dc7df8da255455fb97a0dc79588b839")
}
func (rpc *AvailMockRPC) GetMetadataLatest() (*types.Metadata, error) {
	return types.NewMetadataV10(), nil
}

func (rpc *AvailMockRPC) GetRuntimeVersionLatest() *types.RuntimeVersion {
	return types.NewRuntimeVersion()
}

func (rpc *AvailMockRPC) GetStorageLatest(key types.StorageKey, target *gsrpc_types.AccountInfo) (ok bool, err error) {
	target.Nonce = 10
	return true, nil
}
