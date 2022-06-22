import { ethers } from 'ethers'

const config = {
  submissionInterval: 6,
  genesisOutput: ethers.constants.HashZero,
  historicalBlocks: 0,
  startingBlockNumber: 0,
  startingTimestamp:
    parseInt(process.env.L2OO_STARTING_BLOCK_TIMESTAMP, 10) || Date.now(),
  l2BlockTime: 2,
  sequencerAddress: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
  ownerAddress: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
}

export default config
