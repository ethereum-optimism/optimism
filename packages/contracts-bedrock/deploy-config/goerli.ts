import { ethers } from 'ethers'

const sequencerAddress = '0x6c23a0dcdfc44b7a57bed148de598895e398d984'
const l1StartingBlockTag = ''

const config = {
  submissionInterval: 6,
  genesisOutput: ethers.constants.HashZero,
  historicalBlocks: 0,
  startingBlockNumber: 0,
  l2BlockTime: 2,
  l1StartingBlockTag,
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

  proxyAdmin: '0xe584e1b833ca80020130b1b69f84f90479076168',
  optimismBaseFeeRecipient: '0xf116a24056b647e3211d095c667e951536cdebaa',
  optimismL1FeeRecipient: '0xc731837b696ca3d9720d23336925368ceaa58f83',
  optimismL2FeeRecipient: '0x26862c200bd48c19f39d9e1cd88a3b439611d911',
  outputOracleOwner: '0x6925b8704ff96dee942623d6fb5e946ef5884b63',
  batchSenderAddress: '0xa11d2b908470e17923fff184d48269bebbd9b2a5',
}

export default config
