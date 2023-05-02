import { DeployConfig } from '../src/deploy-config'

// NOTE: The 'mainnet' network is currently being used for bedrock migration rehearsals.
// The system configured below is not yet live on mainnet, and many of the addresses used are
// unsafe for a production system.

// The following addresses are assigned to multiples roles in the system, therfore we save them
// as constants to avoid having to change them in multiple places.
const foundationMultisig = '0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266' // hh test signer 0
const feeRecipient = '0x70997970C51812dc3A010C7d01b50e0d17dc79C8' // hh test signer 1
const mintManager = '0x5C4e7Ba1E219E47948e6e3F55019A647bA501005'

const config: DeployConfig = {
  finalSystemOwner: foundationMultisig,
  controller: foundationMultisig,
  portalGuardian: foundationMultisig,
  proxyAdminOwner: foundationMultisig,

  l1StartingBlockTag:
    '0x85e677d1ebe93fa80bce1ebbf1a0aadbab3433eca4a205260dab39e1fc23b428',
  l1ChainID: 1,
  l2ChainID: 10,
  l2BlockTime: 2,

  maxSequencerDrift: 600,
  sequencerWindowSize: 3600,
  channelTimeout: 300,

  p2pSequencerAddress: '0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65',
  batchInboxAddress: '0xff00000000000000000000000000000000000010',
  batchSenderAddress: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
  l2OutputOracleSubmissionInterval: 1800,
  l2OutputOracleStartingTimestamp: 1683043523,
  l2OutputOracleStartingBlockNumber: 79149121,
  l2OutputOracleProposer: '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
  l2OutputOracleChallenger: foundationMultisig,
  finalizationPeriodSeconds: 3600,

  baseFeeVaultRecipient: feeRecipient,
  l1FeeVaultRecipient: feeRecipient,
  sequencerFeeVaultRecipient: feeRecipient,

  governanceTokenName: 'Optimism',
  governanceTokenSymbol: 'OP',
  governanceTokenOwner: mintManager,

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
