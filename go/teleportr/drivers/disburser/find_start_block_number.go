package disburser

// FilterStartBlockNumberParams holds the arguments passed to
// FindFilterStartBlockNumber.
type FilterStartBlockNumberParams struct {
	// BlockNumber the current block height of the chain.
	BlockNumber uint64

	// NumConfirmations is the number of confirmations required to consider a
	// block final.
	NumConfirmations uint64

	// DeployBlockNumber is the deployment height of the Deposit contract.
	DeployBlockNumber uint64

	// LastProcessedBlockNumber is the height of the last processed block.
	//
	// NOTE: This will be nil on the first invocation, before blocks have been
	// ingested.
	LastProcessedBlockNumber *uint64
}

func (p *FilterStartBlockNumberParams) unconfirmed(blockNumber uint64) bool {
	return p.BlockNumber+1 < blockNumber+p.NumConfirmations
}

// FindFilterStartBlockNumber returns the block height from which to begin
// filtering logs based on the relative heights of the chain, the contract
// deployment, and the last block that was processed.
func FindFilterStartBlockNumber(params FilterStartBlockNumberParams) uint64 {
	// On initilization, always start at the deploy height.
	if params.LastProcessedBlockNumber == nil {
		return params.DeployBlockNumber
	}

	// If the deployment height has not exited the confirmation window, we can
	// still begin our search from the deployment height.
	if params.unconfirmed(params.DeployBlockNumber) {
		return params.DeployBlockNumber
	}

	// Otherwise, start from the block immediately following the last processed
	// block. If that height is still hasn't fully confirmed, we'll use the
	// height of the last confirmed block.
	var filterStartBlockNumber = *params.LastProcessedBlockNumber + 1
	if params.unconfirmed(filterStartBlockNumber) {
		filterStartBlockNumber = params.BlockNumber + 1 - params.NumConfirmations
	}

	return filterStartBlockNumber
}
