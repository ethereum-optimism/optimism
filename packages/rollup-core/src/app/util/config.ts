/* External Imports */
import {add0x} from '@eth-optimism/core-utils'
import {Signer, Wallet} from 'ethers'
import {InfuraProvider, JsonRpcProvider, Provider} from 'ethers/providers'

/* Internal Imports */
import {Environment} from './environment'

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

export const getL1SequencerAddress = (): string => {
  return add0x(Environment.sequencerAddress() || getSequencerWallet().address)
}

let l1DeploymentWallet: Signer
export const getL1DeploymentSigner = (): Signer => {
  getL1Provider().getSigner()
  if (!l1DeploymentWallet) {
    if (!!Environment.l1ContractDeploymentPrivateKey()) {
      l1DeploymentWallet = new Wallet(Environment.l1ContractDeploymentPrivateKey(), getL1Provider())
    } else if (!!Environment.l1ContractDeploymentMnemonic()) {
      l1DeploymentWallet = Wallet.fromMnemonic(Environment.l1ContractDeploymentMnemonic()).connect(getL1Provider())
    } else {
      throw Error('L1 contract deployment private key or mnemonic must be set in order to get L1 Deployment Wallet.')
    }
  }
  return l1DeploymentWallet
}

export const getL1ContractOwnerAddress = async (): Promise<string> => {
  return add0x(Environment.getL1ContractOwnerAddress() || await getL1DeploymentSigner().getAddress())
}
