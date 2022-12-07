import { DeployConfig } from '../src/deploy-config'

const config: DeployConfig = {
  // Core config
  finalSystemOwner: 'DUMMY',
  controller: 'DUMMY',
  l1StartingBlockTag: 'DUMMY',
  l1ChainID: 5,
  l2ChainID: 420,
  l2BlockTime: 2,
  maxSequencerDrift: 1200,
  sequencerWindowSize: 3600,
  channelTimeout: 120,
  p2pSequencerAddress: 'DUMMY',
  batchInboxAddress: '0xff00000000000000000000000000000000000420',
  batchSenderAddress: 'DUMMY',
  l2OutputOracleSubmissionInterval: 20,
  l2OutputOracleStartingTimestamp: -1,
  l2OutputOracleProposer: 'DUMMY',
  l2OutputOracleChallenger: 'DUMMY',
  finalizationPeriodSeconds: 2,

  // L2 network config
  l2GenesisBlockGasLimit: '0x17D7840',
  l2GenesisBlockCoinbase: '0x4200000000000000000000000000000000000011',
  l2GenesisBlockBaseFeePerGas: '0x3b9aca00',
  gasPriceOracleOverhead: 2100,
  gasPriceOracleScalar: 1000000,
}

export default config
