// Number of blocks added to the L2 chain before the first L2 transaction. Genesis are added to the
// chain to initialize the system. However, they create a discrepancy between the L2 block number
// the index of the transaction that corresponds to that block number. For example, if there's 1
// genesis block, then the transaction with an index of 0 corresponds to the block with index 1.
export const NUM_L2_GENESIS_BLOCKS = 1
