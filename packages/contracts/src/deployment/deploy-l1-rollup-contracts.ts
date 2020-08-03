/* Internal Imports */

import { Signer } from 'ethers'
import { AddressResolverMapping, RollupOptions } from './types'
import {
  GAS_LIMIT,
  getL1ContractOwnerAddress,
  getL1DeploymentSigner,
  getL1SequencerAddress,
} from './config'
import { Environment } from './environment'
import { deployAllContracts } from './contract-deploy'

export const deployContracts = async (): Promise<AddressResolverMapping> => {
  const signer: Signer = getL1DeploymentSigner()
  const ownerAddress: string = await getL1ContractOwnerAddress()
  const sequencerAddress: string = getL1SequencerAddress()
  const rollupOptions: RollupOptions = {
    gasLimit: GAS_LIMIT,
    forceInclusionPeriodSeconds: Environment.forceInclusionPeriodSeconds(),
    ownerAddress,
    sequencerAddress,
  }

  return deployAllContracts({
    signer,
    rollupOptions,
  })
}
