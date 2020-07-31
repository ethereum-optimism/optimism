/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, Signer, Wallet, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  RollupDeployConfig,
  AddressResolverMapping,
  deployAllContracts,
  deployAndRegister as originalDeployAndRegister,
} from '../../src'
import { GAS_LIMIT, DEFAULT_FORCE_INCLUSION_PERIOD } from './constants'

export const makeAddressResolver = async (
  wallet: Signer | Wallet
): Promise<AddressResolverMapping> => {
  const [owner, sequencer, l1ToL2TransactionPasser] = await ethers.getSigners()

  const config: RollupDeployConfig = {
    signer: wallet,
    rollupOptions: {
      gasLimit: GAS_LIMIT,
      forceInclusionPeriod: DEFAULT_FORCE_INCLUSION_PERIOD,
      owner: wallet,
      sequencer,
      l1ToL2TransactionPasser,
    },
  }

  return deployAllContracts(config)
}

export const deployAndRegister = async (
  addressResolver: Contract,
  signer: Signer,
  name: string,
  deployConfig: {
    factory: ContractFactory
    params: any[]
  }
): Promise<Contract> => {
  return originalDeployAndRegister(addressResolver, name, {
    ...deployConfig,
    ...{
      signer,
    },
  })
}
