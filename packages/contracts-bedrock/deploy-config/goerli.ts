import { ethers } from 'ethers'

const l1StartingBlockTag =
  '0xafce66a0a2446856112e4069b275ad32b1f4a607888f9c4c59eddf9be81f8670'

const config = {
  // general
  l1StartingBlockTag,
  l1ChainID: 5,
  l2ChainID: 111,
  l2BlockTime: 2,

  // rollup
  maxSequencerDrift: 1000,
  sequencerWindowSize: 120,
  channelTimeout: 120,
  p2pSequencerAddress: '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
  optimismL2FeeRecipient: '0x26862c200bd48c19f39d9e1cd88a3b439611d911',
  batchInboxAddress: '0xff00000000000000000000000000000000000002',
  batchSenderAddress: '0xa11d2b908470e17923fff184d48269bebbd9b2a5',

  // output oracle
  l2OutputOracleSubmissionInterval: 6,
  l2OutputOracleStartingTimestamp: -1, // based on L1 starting tag instead
  l2OutputOracleProposer: '0x6c23a0dcdfc44b7a57bed148de598895e398d984',
  l2OutputOracleOwner: '0x6925b8704ff96dee942623d6fb5e946ef5884b63',

  // l2
  optimismBaseFeeRecipient: '0xf116a24056b647e3211d095c667e951536cdebaa',
  optimismL1FeeRecipient: '0xc731837b696ca3d9720d23336925368ceaa58f83',
  proxyAdmin: '0xe584e1b833ca80020130b1b69f84f90479076168',
  fundDevAccounts: true,

  // deploying
  deploymentWaitConfirmations: 1,
}

export default config
