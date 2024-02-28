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

	avail_Blk_Ref, err := SubmitDataAndWatch(img)

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
func SubmitDataAndWatch(data []byte) (types.AvailBlockRef, error) {

	//Load variables
	var config config.Config
	err := config.GetConfig("../op-avail/config.json")
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

	appID := 0
	// if app id is greater than 0 then it must be created before submitting data
	if AppID != 0 {
		appID = AppID
	}

	c, err := gsrpc_types.NewCall(meta, "DataAvailability.submit_data", gsrpc_types.NewBytes([]byte(data)))
	if err != nil {
		fmt.Printf("cannot create new call: error:%v", err)
		return types.AvailBlockRef{}, err
	}

	// Create the extrinsic
	ext := gsrpc_types.NewExtrinsic(c)

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		fmt.Printf("cannot get block hash: error:%v", err)
		return types.AvailBlockRef{}, err
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		fmt.Printf("cannot get runtime version: error:%v", err)
		return types.AvailBlockRef{}, err
	}

	keyringPair, err := signature.KeyringPairFromSecret(Seed, 42)
	if err != nil {
		fmt.Printf("cannot create LeyPair: error:%v", err)
		return types.AvailBlockRef{}, err
	}

	key, err := gsrpc_types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		fmt.Printf("cannot create storage key: error:%v", err)
		return types.AvailBlockRef{}, err
	}

	var accountInfo gsrpc_types.AccountInfo
	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		fmt.Printf("cannot get latest storage: error:%v", err)
		return types.AvailBlockRef{}, err
	}

	nonce := utils.GetAccountNonce(uint32(accountInfo.Nonce))
	//fmt.Println("Nonce from localDatabase:", nonce, "    ::::::::   from acountInfo:", accountInfo.Nonce)
	o := gsrpc_types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                gsrpc_types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              gsrpc_types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                gsrpc_types.NewUCompactFromUInt(0),
		AppID:              gsrpc_types.NewUCompactFromUInt(uint64(appID)),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(keyringPair, o)
	if err != nil {
		fmt.Printf("cannot sign: error:%v", err)
		return types.AvailBlockRef{}, err
	}

	// Send the extrinsic
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		fmt.Printf("cannot submit extrinsic: error:%v", err)
		return types.AvailBlockRef{}, err
	}

	defer sub.Unsubscribe()
	timeout := time.After(100 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsFinalized {
				return types.AvailBlockRef{BlockHash: string(status.AsFinalized.Hex()), Sender: keyringPair.Address, Nonce: o.Nonce.Int64()}, nil
			}
		case <-timeout:
			return types.AvailBlockRef{}, errors.New("Timitout before getting finalized status")
		}
	}
}

func getSubstrateApiAndMeta(ApiURL string) (*gsrpc.SubstrateAPI, *gsrpc_types.Metadata, error) {
	//Creating new substrate api
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
