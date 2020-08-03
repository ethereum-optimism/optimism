/* External Imports */
import {
  Environment,
  GAS_LIMIT, getL1ContractOwnerAddress, getL1DeploymentSigner,
  getL1SequencerAddress,
} from '@eth-optimism/rollup-core'
import {
  AddressResolverMapping,
  deployAllContracts,
  RollupOptions,
} from '@eth-optimism/rollup-contracts'

import {Signer} from 'ethers'

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

  return deployAllContracts(
    {
      signer,
      rollupOptions,
    }
  )
}