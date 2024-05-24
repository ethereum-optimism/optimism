package scripts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	types "github.com/ethereum-optimism/optimism/op-plasma/cmd/avail/types"
	"github.com/ethereum-optimism/optimism/op-plasma/cmd/avail/utils"
	"github.com/ethereum/go-ethereum/crypto"
)

func SubmitDataAndWatch(specs *types.AvailDASpecs, ctx context.Context, data []byte) (types.AvailBlockRef, error) {
	call, err := createDataAvailabilityCall(specs, data)

	if err != nil {

		return types.AvailBlockRef{}, fmt.Errorf("creating data availability call: %w", err)
	}

	signedExt, sender, nonce, err := prepareAndSignExtrinsic(specs, call)
	if err != nil {
		fmt.Println(err)
		return types.AvailBlockRef{}, fmt.Errorf("preparing and signing extrinsic: %w", err)
	}

	dataCommitment := crypto.Keccak256(data)

	return waitForExtrinsicFinalization(specs, signedExt, sender, nonce, dataCommitment)

}

func prepareAndSignExtrinsic(specs *types.AvailDASpecs, call gsrpc_types.Call) (gsrpc_types.Extrinsic, string, uint32, error) {

	accountInfo, err := fetchAccountInfo(specs)
	if err != nil {
		return gsrpc_types.Extrinsic{}, specs.KeyringPair.Address, 0, err
	}

	nonce := utils.GetAccountNonce(uint32(accountInfo.Nonce))
	ext := gsrpc_types.NewExtrinsic(call)

	err = signExtrinsic(&ext, specs.KeyringPair, specs.GenesisHash, specs.Rv, nonce, specs.AppID)
	if err != nil {
		return gsrpc_types.Extrinsic{}, specs.KeyringPair.Address, nonce, err
	}

	return ext, specs.KeyringPair.Address, nonce, nil
}

func waitForExtrinsicFinalization(specs *types.AvailDASpecs, ext gsrpc_types.Extrinsic, sender string, nonce uint32, dataCommitment []byte) (types.AvailBlockRef, error) {
	sub, err := specs.Api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return types.AvailBlockRef{}, fmt.Errorf("cannot submit extrinsic: %w", err)
	}
	defer sub.Unsubscribe()

	timeout := time.After(specs.Timeout)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				fmt.Printf("Txn inside block %v\n", status.AsInBlock.Hex())
			} else if status.IsFinalized {
				return types.AvailBlockRef{BlockHash: string(status.AsFinalized.Hex()), Sender: sender, Nonce: int64(nonce), Commitment: dataCommitment}, nil
			}
		case <-timeout:
			return types.AvailBlockRef{}, errors.New("timeout before getting finalized status")
		}
	}
}

func fetchAccountInfo(specs *types.AvailDASpecs) (gsrpc_types.AccountInfo, error) {
	var accountInfo gsrpc_types.AccountInfo

	ok, err := specs.Api.RPC.State.GetStorageLatest(specs.StorageKey, &accountInfo)

	if err != nil || !ok {
		return accountInfo, fmt.Errorf("cannot get latest storage: %w", err)
	}

	return accountInfo, nil
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

func createDataAvailabilityCall(specs *types.AvailDASpecs, data []byte) (gsrpc_types.Call, error) {

	c, err := gsrpc_types.NewCall(specs.Meta, "DataAvailability.submit_data", gsrpc_types.NewBytes(data))
	if err != nil {
		return gsrpc_types.Call{}, fmt.Errorf("cannot create new call: %w", err)
	}
	return c, nil
}
