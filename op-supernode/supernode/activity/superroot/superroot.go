package superroot

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	gethlog "github.com/ethereum/go-ethereum/log"
)

// Superroot is an Activity that operates across all chain containers, constructing the superroot at a given timestamp.
type Superroot struct {
	log    gethlog.Logger
	chains map[eth.ChainID]cc.ChainContainer
}

// New creates a new Superroot rpc activity
func New(log gethlog.Logger, chains map[eth.ChainID]cc.ChainContainer) *Superroot {
	return &Superroot{
		log:    log,
		chains: chains,
	}
}

// ActivityName returns the routing name for this activity.
func (s *Superroot) ActivityName() string { return "superroot" }

// RPCAPIs implements RPCActivity by returning a JSON-RPC API namespace for superroot.
func (s *Superroot) RPCNamespace() string    { return "superroot" }
func (s *Superroot) RPCService() interface{} { return &superrootAPI{s: s} }

// superrootAPI hosts JSON-RPC methods for the Superroot activity.
type superrootAPI struct{ s *Superroot }

type atL1Response struct {
	SuperRoot                  eth.Bytes32
	CurrentL1s                 map[eth.ChainID]eth.BlockID
	VerifiedTimestamps         map[eth.ChainID]uint64
	VerifiedTimestampsWithL1   map[eth.ChainID]uint64
	MinCurrentL1               eth.BlockID
	MinVerifiedTimestamp       uint64
	MinVerifiedTimestampWithL1 uint64
}

// AtL1 computes the super-root at the given L1 block number.
func (api *superrootAPI) AtL1(ctx context.Context, blockNumber uint64) (atL1Response, error) {
	return api.s.atL1(ctx, blockNumber)
}

func (s *Superroot) atL1(ctx context.Context, blockNumber uint64) (atL1Response, error) {
	// the superroot is the primary structure that will be returned to the requester.
	// it represents a concatenation of all Chains' output roots, and the shared timestamp
	// and is used to make proposals or to verify proposals for playing Fault Games.
	outputRoots := map[eth.ChainID]eth.Bytes32{}
	var superRoot eth.Bytes32

	// additional information is provided to help the requester make decisions about the superroot.

	// currentL1s shows that the nodes have processed at least up to this L1 block
	// it is checked per Chain, and the minimum L1 block is used.
	currentL1s := map[eth.ChainID]eth.BlockID{}
	var minCurrentL1 eth.BlockID

	// verifiedTimestamps shows that the nodes have verified at least up to this timestamp
	// the Chain Container aggregates the verified timestamps of all Verification Activities and represents it as a single timestamp.
	verifiedTimestamps := map[eth.ChainID]uint64{}
	var minVerifiedTimestamp uint64

	// verifiedTimestampWithL1 shows the highest verified timestamp for a given L1 block
	// the Chain Container aggregates the verified timestamps of all Verification Activities and represents it as a single timestamp.
	verifiedTimestampsWithL1 := map[eth.ChainID]uint64{}
	var minVerifiedTimestampWithL1 uint64

	// construct the superroot
	for chainID, chain := range s.chains {
		if s.log != nil {
			s.log.Debug("fetching output root", "chain_id", chainID.String(), "l1_block", blockNumber)
		}
		out, err := chain.OutputRootAtL1(ctx, blockNumber)
		if err != nil {
			if s.log != nil {
				s.log.Warn("failed to get output root at L1", "chain_id", chainID.String(), "l1_block", blockNumber, "err", err)
			}
			return atL1Response{}, err
		}
		outputRoots[chainID] = out
		if s.log != nil {
			s.log.Debug("got output root", "chain_id", chainID.String(), "l1_block", blockNumber, "output", out)
		}
	}

	// get current l1s
	for chainID, chain := range s.chains {
		if s.log != nil {
			s.log.Debug("query current L1", "chain_id", chainID.String())
		}
		currentL1, err := chain.LastL1(ctx)
		if err != nil {
			if s.log != nil {
				s.log.Warn("failed to get current L1", "chain_id", chainID.String(), "err", err)
			}
			return atL1Response{}, err
		}
		currentL1s[chainID] = currentL1.ID()
		if currentL1.ID().Number < minCurrentL1.Number || minCurrentL1 == (eth.BlockID{}) {
			minCurrentL1 = currentL1.ID()
		}
	}

	// get latest verified timestamps
	for chainID, chain := range s.chains {
		if s.log != nil {
			s.log.Debug("query verified-to timestamp", "chain_id", chainID.String())
		}
		verifiedTimestamp, err := chain.VerifiedToTimestamp()
		if err != nil {
			if s.log != nil {
				s.log.Warn("failed to get verified-to timestamp", "chain_id", chainID.String(), "err", err)
			}
			return atL1Response{}, err
		}
		verifiedTimestamps[chainID] = verifiedTimestamp
		if verifiedTimestamp > minVerifiedTimestamp {
			minVerifiedTimestamp = verifiedTimestamp
		}
	}

	// get latest verified timestamps which exist at the L1 block
	for chainID, chain := range s.chains {
		if s.log != nil {
			s.log.Debug("query verified-to timestamp with L1", "chain_id", chainID.String())
		}
		verifiedTimestampWithL1, err := chain.VerifiedToTimestampWithL1(ctx, blockNumber)
		if err != nil {
			if s.log != nil {
				s.log.Warn("failed to get verified-to timestamp with L1", "chain_id", chainID.String(), "err", err)
			}
			return atL1Response{}, err
		}
		verifiedTimestampsWithL1[chainID] = verifiedTimestampWithL1
		if verifiedTimestampWithL1 > minVerifiedTimestampWithL1 {
			minVerifiedTimestampWithL1 = verifiedTimestampWithL1
		}
	}

	if s.log != nil {
		s.log.Info("superroot.AtL1 done", "l1_block", blockNumber,
			"min_current_l1", minCurrentL1.Number,
			"min_verified_ts", minVerifiedTimestamp,
			"min_verified_ts_with_l1", minVerifiedTimestampWithL1,
		)
	}

	// Build the Superroot bytes32 from chainIDs and output roots per V1 format
	{
		outputs := make([]eth.ChainIDAndOutput, 0, len(outputRoots))
		for chainID, out := range outputRoots {
			outputs = append(outputs, eth.ChainIDAndOutput{ChainID: chainID, Output: out})
		}
		super := eth.NewSuperV1(minVerifiedTimestampWithL1, outputs...)
		superRoot = eth.SuperRoot(super)
		if s.log != nil {
			s.log.Debug("computed superroot", "l1_block", blockNumber, "timestamp", minVerifiedTimestampWithL1, "superroot", superRoot)
		}
	}

	return atL1Response{
		SuperRoot:                  superRoot,
		CurrentL1s:                 currentL1s,
		VerifiedTimestamps:         verifiedTimestamps,
		VerifiedTimestampsWithL1:   verifiedTimestampsWithL1,
		MinCurrentL1:               minCurrentL1,
		MinVerifiedTimestamp:       minVerifiedTimestamp,
		MinVerifiedTimestampWithL1: minVerifiedTimestampWithL1,
	}, nil
}
