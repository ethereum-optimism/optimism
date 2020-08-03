/* External Imports */
import { add0x } from '@eth-optimism/core-utils'
import { ethers, Signer, Wallet } from 'ethers'

/* Internal Imports */
import { Environment } from './environment'

export const GAS_LIMIT = 1_000_000_000

let l1Provider: ethers.providers.Provider
export const getL1Provider = (): ethers.providers.Provider => {
  if (!l1Provider) {
    if (
      !!Environment.l1NodeInfuraNetwork() &&
      !!Environment.l1NodeInfuraProjectId()
    ) {
      l1Provider = new ethers.providers.InfuraProvider(
        Environment.getOrThrow(Environment.l1NodeInfuraNetwork),
        Environment.getOrThrow(Environment.l1NodeInfuraProjectId)
      )
    } else {
      l1Provider = new ethers.providers.JsonRpcProvider(
        Environment.getOrThrow(Environment.l1NodeWeb3Url)
      )
    }
  }
  return l1Provider
}

export const getL1SequencerAddress = (): string => {
  return add0x(Environment.getOrThrow(Environment.sequencerAddress))
}

let l1DeploymentWallet: Signer
export const getL1DeploymentSigner = (): Signer => {
  if (!l1DeploymentWallet) {
    if (!!Environment.l1ContractDeploymentPrivateKey()) {
      l1DeploymentWallet = new Wallet(
        add0x(Environment.l1ContractDeploymentPrivateKey()),
        getL1Provider()
      )
    } else if (!!Environment.l1ContractDeploymentMnemonic()) {
      l1DeploymentWallet = Wallet.fromMnemonic(
        Environment.l1ContractDeploymentMnemonic()
      ).connect(getL1Provider())
    } else {
      throw Error(
        'L1 contract deployment private key or mnemonic must be set in order to get L1 Deployment Wallet.'
      )
    }
  }
  return l1DeploymentWallet
}

export const getL1ContractOwnerAddress = async (): Promise<string> => {
  return add0x(
    Environment.getL1ContractOwnerAddress() ||
      (await getL1DeploymentSigner().getAddress())
  )
}
