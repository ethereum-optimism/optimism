import { ethers } from 'ethers'

const sequencerAddress = '0x08e6e4b77997ce01dfc456e9c1b2c8a74e394b85'
const startingTimestamp = 1658409374

const config = {
  submissionInterval: 6,
  genesisOutput: ethers.constants.HashZero,
  historicalBlocks: 0,
  startingBlockNumber: 0,
  l2BlockTime: 2,
  startingTimestamp,
  sequencerAddress,

  l2CrossDomainMessengerOwner: ethers.constants.AddressZero,
  gasPriceOracleOwner: ethers.constants.AddressZero,
  gasPriceOracleOverhead: 2100,
  gasPriceOracleScalar: 1000000,
  gasPriceOracleDecimals: 6,

  l1BlockInitialNumber: 0,
  l1BlockInitialTimestamp: 0,
  l1BlockInitialBasefee: 10,
  l1BlockInitialHash: ethers.constants.HashZero,
  l1BlockInitialSequenceNumber: 0,

  genesisBlockExtradata: ethers.utils.hexConcat([
    ethers.constants.HashZero,
    sequencerAddress,
    ethers.utils.hexZeroPad('0x', 65),
  ]),
  genesisBlockGasLimit: ethers.BigNumber.from(15000000).toHexString(),

  genesisBlockChainid: 111,
  fundDevAccounts: true,
  p2pSequencerAddress: '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',

  deploymentWaitConfirmations: 1,

  maxSequencerDrift: 1000,
  sequencerWindowSize: 120,
  channelTimeout: 120,

  proxyAdmin: '0x0f6e69f2a9d03c0630e80c022a64b26ec6af6f4d',
  optimismBaseFeeRecipient: '0xe72af8e6ffc820f2936d412f93388d414bc5866a',
  optimismL1FeeRecipient: '0xd1fb02477d488638594bcc35577cf94a4314b125',
  optimismL2FeeRecipient: '0xd6017e83e09f2bce5c739a6032578380cab317cb',
  outputOracleOwner: '0x9e9884bda323929f3b6844c064345e4c39c6cc10',
  batchSenderAddress: '0x85bc55d63c2c1977a553c3b10cf5be3d5a470e79',
}

export default config
