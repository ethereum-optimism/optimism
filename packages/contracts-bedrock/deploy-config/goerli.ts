import { ethers } from 'ethers'

const sequencerAddress = '0x5743191a8a1ffcedfc24f5b7219cb6714df0e5bb'
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

  proxyAdmin: '0x863516d59eefd135485669e14cff3a8fb3836e74',
  optimismBaseFeeRecipient: '0xf3841b313eb0da41d6dd47d82c149dcfa89aafbf',
  optimismL1FeeRecipient: '0xce80bf47c3cc7cf824e917c7b6ff24513b09eba2',
  optimismL2FeeRecipient: '0xd9c09e21b57c98e58a80552c170989b426766aa7',
  outputOracleOwner: '0x7edca314d8e7f3bd7748c2c65f1de12d1a03b780',
  batchSenderAddress: '0x6ec80601358a8297249f20ecf9248a6b16da1aaa',
}

export default config
