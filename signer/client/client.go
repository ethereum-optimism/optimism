package client

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type SignerClient struct {
	client *rpc.Client
	status string
}

func NewSignerClient(endpoint string) (*SignerClient, error) {
	client, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	signer := &SignerClient{client: client}
	// Check if reachable
	version, err := signer.pingVersion()
	if err != nil {
		return nil, err
	}
	signer.status = fmt.Sprintf("ok [version=%v]", version)
	return signer, nil
}

func (s *SignerClient) pingVersion() (string, error) {
	var v string
	if err := s.client.Call(&v, "health_status"); err != nil {
		return "", err
	}
	return v, nil
}

type SignTransactionResult struct {
	Signature hexutil.Bytes `json:"signature"`
}

func (s *SignerClient) SignTransaction(
	ctx context.Context,
	tx *types.Transaction,
) (*types.Transaction, error) {

	txraw, err := tx.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal transaction")
	}

	signer := types.LatestSignerForChainID(tx.ChainId())
	digest := signer.Hash(tx).Hex()

	result := SignTransactionResult{}
	if err := s.client.Call(&result, "signer_signTransaction", hexutil.Encode(txraw), digest); err != nil {
		return nil, errors.Wrap(err, "signer_signTransaction failed")
	}

	tx.WithSignature(signer, result.Signature)

	return tx, nil
}
