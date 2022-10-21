/**
 * Predeploys are Solidity contracts that are injected into the initial L2 state and provide
 * various useful functions.
 * Notes:
 * 0x42...04 was the address of the OVM_ProxySequencerEntrypoint. This contract is no longer in
 * use and has therefore been removed. We may place a new predeployed contract at this address
 * in the future. See https://github.com/ethereum-optimism/optimism/pull/549 for more info.
 */
export const predeploys = {
  L2ToL1MessagePasser: '0x4200000000000000000000000000000000000016',
  DeployerWhitelist: '0x4200000000000000000000000000000000000002',
  L2CrossDomainMessenger: '0x4200000000000000000000000000000000000007',
  GasPriceOracle: '0x420000000000000000000000000000000000000F',
  L2StandardBridge: '0x4200000000000000000000000000000000000010',
  SequencerFeeVault: '0x4200000000000000000000000000000000000011',
  OptimismMintableERC20Factory: '0x4200000000000000000000000000000000000012',
  L1BlockNumber: '0x4200000000000000000000000000000000000013',
  L1Block: '0x4200000000000000000000000000000000000015',
  LegacyERC20ETH: '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',
  WETH9: '0x4200000000000000000000000000000000000006',
  GovernanceToken: '0x4200000000000000000000000000000000000042',
  LegacyMessagePasser: '0x4200000000000000000000000000000000000000',
  L2ERC721Bridge: '0x4200000000000000000000000000000000000014',
  OptimismMintableERC721Factory: '0x4200000000000000000000000000000000000017',
  ProxyAdmin: '0x4200000000000000000000000000000000000018',
}

export const futurePredeploys = {
  System1: '0x4200000000000000000000000000000000000014',
}
