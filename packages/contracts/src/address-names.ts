type addrNames = {
  [key: string]: string
}

// todo: split up
/**
 * This object defines the correct names to be used in the Address Manager and deployment artifacts.
 */
export const addressNames: addrNames = {
  addressManager: 'Lib_AddressManager',
  sequencer: 'OVM_Sequencer',
  proposer: 'OVM_Proposer',
  addressDictator: 'AddressDictator',
  chugsplashDictator: 'ChugsplashDictator',
  storageContainerCtc: 'ChainStorageContainer-CTC-batches',
  storageContainerScc: 'ChainStorageContainer-SCC-batches',
  canonicalTransactionChain: 'CanonicalTransactionChain',
  stateCommitmentChain: 'StateCommitmentChain',
  bondManager: 'BondManager',
  implL1CrossDomainMessenger: 'OVM_L1CrossDomainMessenger',
  proxyL1CrossDomainMessenger: 'Proxy__OVM_L1CrossDomainMessenger',
  proxyL1StandardBridge: 'Proxy__OVM_L1StandardBridge',
}
