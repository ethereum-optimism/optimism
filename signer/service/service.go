package service

import (
	"github.com/ethereum/go-ethereum/log"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/rpc"
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
