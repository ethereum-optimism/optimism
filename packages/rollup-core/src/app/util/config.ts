/* External Imports */
import { InfuraProvider, JsonRpcProvider, Provider } from 'ethers/providers'
import { Wallet } from 'ethers'

/* Internal Imports */
import { Environment } from './environment'

let l1Provider: JsonRpcProvider
export const getL1Provider = (): JsonRpcProvider => {
  if (!l1Provider) {
    if (
      !!Environment.l1NodeInfuraNetwork() &&
      !!Environment.l1NodeInfuraProjectId()
    ) {
      l1Provider = new InfuraProvider(
        Environment.getOrThrow(Environment.l1NodeInfuraNetwork),
        Environment.getOrThrow(Environment.l1NodeInfuraProjectId)
      )
    } else {
      l1Provider = new JsonRpcProvider(
        Environment.getOrThrow(Environment.l1NodeWeb3Url)
      )
    }
  }
  return l1Provider
}

let l2Provider: Provider
export const getL2Provider = (): Provider => {
  if (!l2Provider) {
    l2Provider = new JsonRpcProvider(
      Environment.getOrThrow(Environment.l2NodeWeb3Url)
    )
  }
  return l2Provider
}

let submitToL2GethWallet: Wallet
export const getSubmitToL2GethWallet = (): Wallet => {
  if (!submitToL2GethWallet) {
    submitToL2GethWallet = new Wallet(
      Environment.getOrThrow(Environment.submitToL2GethPrivateKey),
      getL2Provider()
    )
  }
  return submitToL2GethWallet
}

let sequencerWallet: Wallet
export const getSequencerWallet = (): Wallet => {
  if (!sequencerWallet) {
    sequencerWallet = new Wallet(
      Environment.getOrThrow(Environment.sequencerPrivateKey),
      getL1Provider()
    )
  }
  return sequencerWallet
}

let stateRootSubmissionWallet: Wallet
export const getStateRootSubmissionWallet = (): Wallet => {
  if (!stateRootSubmissionWallet) {
    stateRootSubmissionWallet = new Wallet(
      Environment.getOrThrow(Environment.stateRootSubmissionWallet),
      getL1Provider()
    )
  }
  return stateRootSubmissionWallet
}
