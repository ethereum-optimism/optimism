import { ethers } from 'ethers'

const { env } = process

const startingTimestamp =
  typeof env.L2OO_STARTING_BLOCK_TIMESTAMP === 'string'
    ? ethers.BigNumber.from(env.L2OO_STARTING_BLOCK_TIMESTAMP).toNumber()
    : Date.now() / 1000

const config = {
  submissionInterval: 6,
  genesisOutput: ethers.constants.HashZero,
  historicalBlocks: 0,
  startingBlockNumber: 0,
  l2BlockTime: 2,

  startingTimestamp,
  sequencerAddress: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',

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

  proxyAdmin: '0x829BD824B016326A401d083B33D092293333A830',
  genesisBlockExtradata: ethers.utils.hexConcat([
    ethers.constants.HashZero,
    '0xca062b0fd91172d89bcd4bb084ac4e21972cc467',
    ethers.utils.hexZeroPad('0x', 65),
  ]),
  genesisBlockGasLimit: ethers.BigNumber.from(15000000).toHexString(),

  genesisBlockChainid: 901,
  fundDevAccounts: true,
  optimsismBaseFeeRecipient: '0xBcd4042DE499D14e55001CcbB24a551F3b954096',
  optimismL1FeeRecipient: '0x71bE63f3384f5fb98995898A86B02Fb2426c5788',

  deploymentWaitConfirmations: 1,

  maxSequencerDrift: 10,
  sequencerWindowSize: 2,

  ownerAddress: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
}

export default config
