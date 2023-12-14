package clients

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-ufm/pkg/metrics"
	"github.com/ethereum/go-ethereum/common"

	signer "github.com/ethereum-optimism/optimism/op-service/signer"
	optls "github.com/ethereum-optimism/optimism/op-service/tls"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type InstrumentedSignerClient struct {
	c            *signer.SignerClient
	providerName string
}

func NewSignerClient(providerName string, logger log.Logger, endpoint string, tlsConfig optls.CLIConfig) (*InstrumentedSignerClient, error) {
	start := time.Now()
	c, err := signer.NewSignerClient(logger, endpoint, tlsConfig)
	if err != nil {
		metrics.RecordErrorDetails(providerName, "signer.NewSignerClient", err)
		return nil, err
	}
	metrics.RecordRPCLatency(providerName, "signer", "NewSignerClient", time.Since(start))
	return &InstrumentedSignerClient{c: c, providerName: providerName}, nil
}

func (i *InstrumentedSignerClient) SignTransaction(ctx context.Context, chainId *big.Int, from *common.Address, tx *types.Transaction) (*types.Transaction, error) {
	start := time.Now()
	tx, err := i.c.SignTransaction(ctx, chainId, *from, tx)
	if err != nil {
		metrics.RecordErrorDetails(i.providerName, "signer.SignTransaction", err)
		return nil, err
	}
	metrics.RecordRPCLatency(i.providerName, "signer", "SignTransaction", time.Since(start))
	return tx, err
}
