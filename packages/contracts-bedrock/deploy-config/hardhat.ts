import { ethers } from 'ethers'

const { env } = process

const l1GenesisBlockTimestamp =
  typeof env.L1_GENESIS_TIMESTAMP === 'string'
    ? ethers.BigNumber.from(env.L1_GENESIS_TIMESTAMP).toNumber()
    : Math.floor(Date.now() / 1000)

const config = {
  // general
  l1StartingBlockTag: 'earliest',
  l1ChainID: 901,
  l2ChainID: 902,
  l2BlockTime: 2,

  // rollup
  maxSequencerDrift: 10,
  sequencerWindowSize: 4,
  channelTimeout: 40,
  p2pSequencerAddress: '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
  optimismL2FeeRecipient: '0xd9c09e21b57c98e58a80552c170989b426766aa7',
  batchInboxAddress: '0xff00000000000000000000000000000000000000',
  batchSenderAddress: '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',

  // output oracle
  l2OutputOracleSubmissionInterval: 6,
  l2OutputOracleStartingTimestamp: -1, // based on L1 starting tag instead
  l2OutputOracleProposer: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
  l2OutputOracleOwner: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',

  // l1
  l1GenesisBlockTimestamp,

  // l2
  proxyAdmin: ethers.constants.AddressZero,
  fundDevAccounts: true,

  // deploying
  deploymentWaitConfirmations: 1,
}

export default config
