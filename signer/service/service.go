package service

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type SignerService struct {
	logger log.Logger
}

func NewSignerService(l log.Logger) SignerService {
	return SignerService{logger: l}
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
	keyName string, txraw hexutil.Bytes, digest hexutil.Bytes,
) (SignTransactionResponse, error) {

	tx := types.Transaction{}
	if err := tx.UnmarshalBinary(txraw); err != nil {
		return SignTransactionResponse{}, new(TransactionUnmarshalError)
	}

	signer := types.LatestSignerForChainID(tx.ChainId())
	if signer.Hash(&tx).Hex() != digest.String() {
		return SignTransactionResponse{}, new(InvalidDigestError)
	}

	signature := "signature"

	s.logger.Info(
		"signed transaction",
		"keyname", keyName,
		"digest", digest,
		"client.name", "client",
		"tx.raw", txraw,
		"tx.value", tx.Value(),
		"tx.to", tx.To().Hex(),
		"tx.nonce", tx.Nonce(),
		"tx.gas", tx.Gas(),
		"tx.gasprice", tx.GasPrice(),
		"tx.hash", tx.Hash().Hex(),
		"tx.chainid", tx.ChainId(),
		"signature", signature,
	)

	return SignTransactionResponse{Signature: signature}, nil
}
