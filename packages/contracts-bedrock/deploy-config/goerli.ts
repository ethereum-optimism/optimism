import { ethers } from 'ethers'

const sequencerAddress = '0x0631f9bccb86548dc4a574c730a46d6ca283a338'
const startingTimestamp = 1656654016

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

  proxyAdmin: '0x05e22b779967b86fb9572e8292090be2d5c1cab7',
  optimismBaseFeeRecipient: '0xec4f588262821a7c1f722e5bc40dc5332335c47f',
  optimismL1FeeRecipient: '0x8fd8d6b9e556cf4791ff9c99a56420ac2fdd2b59',
  optimismL2FeeRecipient: '0x7890eee9efd42496c63f3ec71bf61bf96af088d0',
  outputOracleOwner: '0x0f01ce071078396040a4a0de613aa024aba2d18f',
  batchSenderAddress: '0x32b317fc8d35e015cd9942bc9c7cecaf7f651838',
}

export default config
