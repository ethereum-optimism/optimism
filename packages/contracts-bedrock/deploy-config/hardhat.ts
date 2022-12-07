import { DeployConfig } from '../src/deploy-config'

const config: DeployConfig = {
  // Core config
  numDeployConfirmations: 1,
  finalSystemOwner: '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
  l1StartingBlockTag: 'earliest',
  l1ChainID: 900,
  l2ChainID: 901,
  l2BlockTime: 2,
  maxSequencerDrift: 300,
  sequencerWindowSize: 15,
  channelTimeout: 40,
  p2pSequencerAddress: '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
  batchInboxAddress: '0xff00000000000000000000000000000000000000',
  batchSenderAddress: '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
  l2OutputOracleSubmissionInterval: 6,
  l2OutputOracleStartingTimestamp: -1,
  l2OutputOracleProposer: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
  l2OutputOracleChallenger: '0x6925B8704Ff96DEe942623d6FB5e946EF5884b63',
  finalizationPeriodSeconds: 2,

  // L2 network config
  l2GenesisBlockBaseFeePerGas: '0x3B9ACA00',
}

export default config
