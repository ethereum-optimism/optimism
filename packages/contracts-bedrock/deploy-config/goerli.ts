import { ethers } from 'ethers'

const sequencerAddress = '0x884ff7da19e669c0f3ffa9a73d480f55fca3209c'
const startingTimestamp = 1657827243

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

  proxyAdmin: '0xf1d43756128f3a0cceb6d830f5b15639b662794d',
  optimismBaseFeeRecipient: '0x8ab8e0f1c9ac841219a55f956c8ed7e20051b55c',
  optimismL1FeeRecipient: '0x5b88072b02a194b75c0f7cc2514056f093e76a7b',
  optimismL2FeeRecipient: '0xf03ac5d0435f54019b83e427c1f6e4dafdd20c39',
  outputOracleOwner: '0xeed5a2c7cc3c5a4573944427530d15e37afd5460',
  batchSenderAddress: '0x8d73fa85fdc849fca19820152b21e5a6f4008ab6',
}

export default config
