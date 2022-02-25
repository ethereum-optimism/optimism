/**
 * Predeploys are Solidity contracts that are injected into the initial L2 state and provide
 * various useful functions.
 *
 * Notes:
 * 0x42...04 was the address of the OVM_ProxySequencerEntrypoint. This contract is no longer in
 * use and has therefore been removed. We may place a new predeployed contract at this address
 * in the future. See https://github.com/ethereum-optimism/optimism/pull/549 for more info.
 */
export const predeploys = {
  OVM_L2ToL1MessagePasser: '0x4200000000000000000000000000000000000000',
  OVM_DeployerWhitelist: '0x4200000000000000000000000000000000000002',
  L2CrossDomainMessenger: '0x4200000000000000000000000000000000000007',
  OVM_GasPriceOracle: '0x420000000000000000000000000000000000000F',
  L2StandardBridge: '0x4200000000000000000000000000000000000010',
  OVM_SequencerFeeVault: '0x4200000000000000000000000000000000000011',
  L2StandardTokenFactory: '0x4200000000000000000000000000000000000012',
  OVM_L1BlockNumber: '0x4200000000000000000000000000000000000013',

  // We're temporarily disabling OVM_ETH because the jury is still out on whether or not ETH as an
  // ERC20 is desirable.
  OVM_ETH: '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',

  // We're also putting WETH9 at the old OVM_ETH address.
  WETH9: '0x4200000000000000000000000000000000000006',
}

export const futurePredeploys = {
  // System addresses, for use later
  System0: '0x4200000000000000000000000000000000000042',
  System1: '0x4200000000000000000000000000000000000014',
}
