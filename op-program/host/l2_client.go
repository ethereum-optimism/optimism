package host

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/sources/caching"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type L2Client struct {
	*sources.L2Client

	// l2Head is the L2 block hash that we use to fetch L2 output
	l2Head common.Hash
}

type L2ClientConfig struct {
	*sources.L2ClientConfig
	L2Head common.Hash
}

func NewL2Client(client client.RPC, log log.Logger, metrics caching.Metrics, config *L2ClientConfig) (*L2Client, error) {
	l2Client, err := sources.NewL2Client(client, log, metrics, config.L2ClientConfig)
	if err != nil {
		return nil, err
	}
	return &L2Client{
		L2Client: l2Client,
		l2Head:   config.L2Head,
	}, nil
}

func (s *L2Client) OutputByRoot(ctx context.Context, l2OutputRoot common.Hash) (eth.Output, error) {
	output, err := s.outputAtBlock(ctx, s.l2Head)
	if err != nil {
		return nil, err
	}
	if eth.OutputRoot(output) != eth.Bytes32(l2OutputRoot) {
		// For fault proofs, we only reference outputs at the l2 head at boot time
		// The caller shouldn't be requesting outputs at any other block
		return nil, fmt.Errorf("unknown output root")
	}
	return output, nil
}

func (s *L2Client) outputAtBlock(ctx context.Context, blockHash common.Hash) (eth.Output, error) {
	head, err := s.InfoByHash(ctx, blockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get L2 block by hash: %w", err)
	}
	if head == nil {
		return nil, ethereum.NotFound
	}

	proof, err := s.GetProof(ctx, predeploys.L2ToL1MessagePasserAddr, []common.Hash{}, blockHash.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get contract proof at block %s: %w", blockHash, err)
	}
	if proof == nil {
		return nil, fmt.Errorf("proof %w", ethereum.NotFound)
	}
	// make sure that the proof (including storage hash) that we retrieved is correct by verifying it against the state-root
	if err := proof.Verify(head.Root()); err != nil {
		return nil, fmt.Errorf("invalid withdrawal root hash, state root was %s: %w", head.Root(), err)
	}
	stateRoot := head.Root()
	return &eth.OutputV0{
		StateRoot:                eth.Bytes32(stateRoot),
		MessagePasserStorageRoot: eth.Bytes32(proof.StorageHash),
		BlockHash:                blockHash,
	}, nil
}
