import { ethers } from 'ethers'

const config = {
  submissionInterval: 6,
  l2BlockTime: 2,
  genesisOutput: ethers.constants.HashZero,
  historicalBlocks: 0,
  startingTimestamp: 1652907966,
  sequencerAddress: '0x7431310e026B69BFC676C0013E12A1A11411EEc9',
}

export default config
