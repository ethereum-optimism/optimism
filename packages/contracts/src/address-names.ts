type addrNames = {
  [key: string]: string
}

/**
 * For contracts and accounts listed in the Address Manager, this object defines the correct names to be used in
 * the Address Manager and deployment artifacts.
 */
export const managedNames: addrNames = {
  storageContainerCtc: 'ChainStorageContainer-CTC-batches',
  storageContainerScc: 'ChainStorageContainer-SCC-batches',
  canonicalTransactionChain: 'CanonicalTransactionChain',
  stateCommitmentChain: 'StateCommitmentChain',
  bondManager: 'BondManager',
  implL1CrossDomainMessenger: 'OVM_L1CrossDomainMessenger',
  proxyL1CrossDomainMessenger: 'Proxy__OVM_L1CrossDomainMessenger',
  proxyL1StandardBridge: 'Proxy__OVM_L1StandardBridge',
}

/**
 * For contracts not listed in the Address Manager, this object defines the correct names to be used
 * in deployment artifacts.
 */
export const unmanagedNames: addrNames = {
  addressDictator: 'AddressDictator',
  chugsplashDictator: 'ChugsplashDictator',
  addressManager: 'Lib_AddressManager',
  sequencer: 'OVM_Sequencer',
  proposer: 'OVM_Proposer',
}
