import { DeployConfig } from '../src/deploy-config'

// NOTE: The 'mainnet' network is currently being used for bedrock migration rehearsals.
// The system configured below is not yet live on mainnet, and many of the addresses used are
// unsafe for a production system.
const config: DeployConfig = {
  finalSystemOwner: '0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A',
  controller: '0x78339d822c23d943e4a2d4c3dd5408f66e6d662d',
  portalGuardian: '0x78339d822c23d943e4a2d4c3dd5408f66e6d662d',

  l1StartingBlockTag:
    '0x126e52a0cc0ae18948f567ee9443f4a8f0db67c437706e35baee424eb314a0d0',
  l1ChainID: 1,
  l2ChainID: 10,
  l2BlockTime: 2,

  maxSequencerDrift: 600,
  sequencerWindowSize: 3600,
  channelTimeout: 300,

  p2pSequencerAddress: '0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65',
  batchInboxAddress: '0xff00000000000000000000000000000000000010',
  batchSenderAddress: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
  l2OutputOracleSubmissionInterval: 20,
  l2OutputOracleStartingTimestamp: 1679069195,
  l2OutputOracleStartingBlockNumber: 79149704,
  l2OutputOracleProposer: '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
  l2OutputOracleChallenger: '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
  finalizationPeriodSeconds: 2,

  proxyAdminOwner: '0x90F79bf6EB2c4f870365E785982E1f101E93b906',
  baseFeeVaultRecipient: '0x90F79bf6EB2c4f870365E785982E1f101E93b906',
  l1FeeVaultRecipient: '0x90F79bf6EB2c4f870365E785982E1f101E93b906',
  sequencerFeeVaultRecipient: '0x90F79bf6EB2c4f870365E785982E1f101E93b906',

  governanceTokenName: 'Optimism',
  governanceTokenSymbol: 'OP',
  governanceTokenOwner: '0x90F79bf6EB2c4f870365E785982E1f101E93b906',

  l2GenesisBlockGasLimit: '0x17D7840',
  l2GenesisBlockCoinbase: '0x4200000000000000000000000000000000000011',
  l2GenesisBlockBaseFeePerGas: '0x3b9aca00',

  gasPriceOracleOverhead: 2100,
  gasPriceOracleScalar: 1000000,
  eip1559Denominator: 50,
  eip1559Elasticity: 10,

  l2GenesisRegolithTimeOffset: '0x0',
}

export default config
