import { ethers } from 'ethers'

const { env } = process

const l1GenesisTimestamp =
  typeof env.L1_GENESIS_TIMESTAMP === 'string'
    ? ethers.BigNumber.from(env.L1_GENESIS_TIMESTAMP).toNumber()
    : Math.floor(Date.now() / 1000)

const config = {
  submissionInterval: 6,
  genesisOutput: ethers.constants.HashZero,
  historicalBlocks: 0,
  startingBlockNumber: 0,
  l1StartingBlockTag: 'earliest',
  l2BlockTime: 2,
  l1GenesisTimestamp,
  sequencerAddress: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
  maxSequencerDrift: 10,
  sequencerWindowSize: 4,
  channelTimeout: 40,
  outputOracleOwner: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
  fundDevAccounts: true,
}

export default config
