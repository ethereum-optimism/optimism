package service

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type SignerService struct {
	logger   log.Logger
	provider SignatureProvider
}

func NewSignerService(logger log.Logger) SignerService {
	return SignerService{
		logger:   logger,
		provider: NewKMSClient(logger),
	}
}

func (s *SignerService) RegisterAPIs(server *oprpc.Server) {
	server.AddAPI(rpc.API{
		Namespace: "signer",
		Service:   s,
	})
}

type SignTransactionResponse struct {
	Signature string `json:"signature"`
}

func (s *SignerService) SignTransaction(
	ctx context.Context, txraw hexutil.Bytes, digest hexutil.Bytes,
) (*SignTransactionResponse, error) {

	// TODO: fix hardcoded key name
	keyName := "projects/op-dev-signer/locations/nam6/keyRings/signer/cryptoKeys/zhwrd-test-key/cryptoKeyVersions/1"
	clientName := "client"

	tx := types.Transaction{}
	if err := tx.UnmarshalBinary(txraw); err != nil {
		return nil, new(TransactionUnmarshalError)
	}

	signer := types.LatestSignerForChainID(tx.ChainId())
	expectedDigest := signer.Hash(&tx).Hex()
	if expectedDigest != digest.String() {
		return nil, new(InvalidDigestError)
	}

	signature, err := s.provider.Sign(ctx, keyName, digest)
	if err != nil {
		s.logger.Error("signature error", "error", err)
		return nil, err
	}

	s.logger.Info(
		"signed transaction",
		"digest", digest,
		"client.name", clientName,
		"keyname", keyName,
		"tx.raw", txraw,
		"tx.value", tx.Value(),
		"tx.to", tx.To().Hex(),
		"tx.nonce", tx.Nonce(),
		"tx.gas", tx.Gas(),
		"tx.gasprice", tx.GasPrice(),
		"tx.hash", tx.Hash().Hex(),
		"tx.chainid", tx.ChainId(),
		"signature", hexutil.Encode(signature),
	)

	return &SignTransactionResponse{Signature: hexutil.Encode(signature)}, nil
}
