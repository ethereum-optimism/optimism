package types

import (
	"encoding/json"
	"fmt"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum-optimism/optimism/op-plasma/cmd/avail/utils"
	"github.com/ethereum/go-ethereum/log"
)

type AvailBlockRef struct {
	BlockHash  string // Hash for block on avail chain
	Sender     string // sender address to filter extrinsic out sepecifically for this address
	Nonce      int64  // nonce to filter specific extrinsic
	Commitment []byte
}

func (a *AvailBlockRef) MarshalToBinary() ([]byte, error) {
	ref_bytes, err := json.Marshal(a)
	if err != nil {
		return []byte{}, fmt.Errorf("unable to covert the avail block referece into array of bytes and getting error:%v", err)
	}
	return ref_bytes, nil
}

func (a *AvailBlockRef) UnmarshalFromBinary(avail_blk_Ref []byte) error {
	err := json.Unmarshal(avail_blk_Ref, a)
	if err != nil {
		return fmt.Errorf("unable to convert avail_blk_Ref bytes to AvailBlockRef Struct and getting error:%v", err)
	}
	return nil
}

type AvailDASpecs struct {
	Timeout     time.Duration
	AppID       int
	Api         *gsrpc.SubstrateAPI
	Meta        *gsrpc_types.Metadata
	GenesisHash gsrpc_types.Hash
	Rv          *gsrpc_types.RuntimeVersion
	KeyringPair signature.KeyringPair
	StorageKey  gsrpc_types.StorageKey
}

func NewAvailDASpecs(ApiURL string, AppID int, Seed string, Timeout time.Duration) (*AvailDASpecs, error) {
	AppID = utils.EnsureValidAppID(AppID)
	api, _ := utils.GetSubstrateApi(ApiURL)

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		log.Warn("⚠️ cannot get metadata: error:%v", err)
		return nil, err
	}

	genesisHash, rv, err := utils.FetchChainData(api)
	if err != nil {
		return nil, err
	}

	keyringPair, err := signature.KeyringPairFromSecret(Seed, 42)
	if err != nil {
		log.Warn("⚠️ cannot create LeyPair: error:%v", err)
		return nil, err
	}

	storageKey, err := gsrpc_types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		log.Warn("⚠️ cannot create storage key: error:%v", err)
		return nil, err
	}

	return &AvailDASpecs{
		Timeout:     Timeout,
		AppID:       AppID,
		Api:         api,
		Meta:        meta,
		GenesisHash: genesisHash,
		Rv:          rv,
		KeyringPair: keyringPair,
		StorageKey:  storageKey,
	}, nil
}
