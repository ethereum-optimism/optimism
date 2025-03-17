package signer

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type RemoteSigner struct {
	client *SignerClient
	sender *common.Address
}

var _ BlockSigner = (*RemoteSigner)(nil)

func NewRemoteSigner(logger log.Logger, config CLIConfig) (*RemoteSigner, error) {
	signerClient, err := NewSignerClientFromConfig(logger, config)
	if err != nil {
		return nil, err
	}
	senderAddress := common.HexToAddress(config.Address)
	return &RemoteSigner{signerClient, &senderAddress}, nil
}

func (s *RemoteSigner) SignBlockV1(ctx context.Context, chainID eth.ChainID, payloadHash common.Hash) (sig eth.Bytes65, err error) {
	if s.client == nil {
		return eth.Bytes65{}, errors.New("signer is closed")
	}

	// We use API V1 for now, since the server may not support V2 yet
	blockPayloadArgs := &BlockPayloadArgs{
		Domain:        SigningDomainBlocksV1,
		ChainID:       chainID.ToBig(),
		PayloadHash:   payloadHash[:],
		SenderAddress: s.sender,
	}
	signature, err := s.client.SignBlockPayload(ctx, blockPayloadArgs)
	if err != nil {
		return eth.Bytes65{}, err
	}
	return signature, nil
}

func (s *RemoteSigner) Close() error {
	if s.client != nil {
		s.client.Close()
	}
	s.client = nil
	return nil
}
