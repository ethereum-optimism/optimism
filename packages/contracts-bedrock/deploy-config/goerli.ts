import { ethers } from 'ethers'

const sequencerAddress = '0xe4aef353bd01802ae668faffa793df8ec391f722'
const startingTimestamp = 1658160349

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

  proxyAdmin: '0x7bc8b30aa3495b41b17d24b828a3129e2dced692',
  optimismBaseFeeRecipient: '0x8889b2e83d26ded86496df023cf8aeafc38d1ef7',
  optimismL1FeeRecipient: '0xa09558a4c7707b088f725e67e2462bbd0e5052b0',
  optimismL2FeeRecipient: '0xdefd40e85a70bc75f5d35adfbf2ac30b148920c9',
  outputOracleOwner: '0x7f9220322feae82aee2a8c139e77b5b4e93d7bf1',
  batchSenderAddress: '0xf773ba58cbd808e7ad92d0a9933186ad840322e3',
}

export default config
