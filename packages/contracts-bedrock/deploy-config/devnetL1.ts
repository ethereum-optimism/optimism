import { DeployConfig } from '../src/deploy-config'

const config: DeployConfig = {
  // Core config
  finalSystemOwner: '0xBcd4042DE499D14e55001CcbB24a551F3b954096',
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
  l2OutputOracleSubmissionInterval: 20,
  l2OutputOracleStartingTimestamp: -1,
  l2OutputOracleProposer: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
  l2OutputOracleChallenger: '0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65',
  finalizationPeriodSeconds: 2,

  // L2 network config
  l2GenesisBlockGasLimit: '0xE4E1C0',
  l2GenesisBlockCoinbase: '0x42000000000000000000000000000000000000f0',
  l2GenesisBlockBaseFeePerGas: '0x3B9ACA00',
  gasPriceOracleOverhead: 2100,
  gasPriceOracleScalar: 1000000,

  // L1 network config
  l1BlockTime: 15,
  cliqueSignerAddress: '0xca062b0fd91172d89bcd4bb084ac4e21972cc467',
}

export default config
