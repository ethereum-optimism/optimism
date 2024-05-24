package scripts

import (
	"fmt"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	"github.com/ethereum-optimism/optimism/op-plasma/cmd/avail/types"
	"github.com/vedhavyas/go-subkey"
)

func GetBlockExtrinsicData(specs types.AvailDASpecs, avail_blk_ref types.AvailBlockRef) ([]byte, error) {

	Hash := avail_blk_ref.BlockHash
	Address := avail_blk_ref.Sender
	Nonce := avail_blk_ref.Nonce

	avail_blk, err := fetchBlock(specs.Api, Hash)
	if err != nil {
		panic(fmt.Sprintf("cannot fetch block: %v", err))
	}

	return extractExtrinsic(Address, Hash, Nonce, avail_blk)
}

func fetchBlock(api *gsrpc.SubstrateAPI, Hash string) (*gsrpc_types.SignedBlock, error) {

	blk_hash, err := gsrpc_types.NewHashFromHexString(Hash)

	if err != nil {
		return &gsrpc_types.SignedBlock{}, fmt.Errorf("unable to convert string hash into types.hash, error:%v", err)
	}

	avail_blk, err := api.RPC.Chain.GetBlock(blk_hash)
	if err != nil {
		return &gsrpc_types.SignedBlock{}, fmt.Errorf("cannot get block for hash:%v and getting error:%v", Hash, err)
	}

	return avail_blk, nil
}

func extractExtrinsic(Address string, Hash string, Nonce int64, avail_blk *gsrpc_types.SignedBlock) ([]byte, error) {

	for _, ext := range avail_blk.Block.Extrinsics {

		ext_Addr, err := subkey.SS58Address(ext.Signature.Signer.AsID.ToBytes(), 42)
		if err != nil {
			fmt.Println("unable to get sender address from extrinsic", "err", err)
		}

		if ext_Addr == Address && ext.Signature.Nonce.Int64() == Nonce {
			args := ext.Method.Args
			var data []byte
			err = codec.Decode(args, &data)
			if err != nil {
				return []byte{}, fmt.Errorf("unable to decode the extrinsic data by address: %v with nonce: %v", Address, Nonce)
			}

			return data, nil
		}
	}

	return []byte{}, fmt.Errorf("didn't find any extrinsic data for address:%v in block having hash:%v", Address, Hash)
}
