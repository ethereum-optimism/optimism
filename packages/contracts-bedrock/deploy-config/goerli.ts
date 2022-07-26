import { ethers } from 'ethers'

const sequencerAddress = '0x15646e889e758990edef3faa1ff01444764b4890'
const startingTimestamp = 1658861222

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

  proxyAdmin: '0x75d82c3c102305412c13679467a11a8017bcdc7d',
  optimismBaseFeeRecipient: '0xdf85652fb32d2b4ed7915636bfbabfda9075db1c',
  optimismL1FeeRecipient: '0xb86af6383a4c64826dcc3212fb435f8233587923',
  optimismL2FeeRecipient: '0x36bdede0099a1079345bafa47db609225acf6c37',
  outputOracleOwner: '0x3d6997b9c0e853905971ceb33091d267abb154a0',
  batchSenderAddress: '0xc569fb3062658ae31e9c7ece71cc5f40ff5404c4',
}

export default config
