import { Contract } from 'ethers'
import { JsonRpcProvider } from '@ethersproject/providers'
import { getContractInterface } from '@eth-optimism/contracts'

import { ZERO_ADDRESS } from './constants'

export const loadContract = (
  name: string,
  address: string,
  provider: JsonRpcProvider
): Contract => {
  return new Contract(address, getContractInterface(name) as any, provider)
}

export const loadContractFromManager = async (
  name: string,
  Lib_AddressManager: Contract,
  provider: JsonRpcProvider
): Promise<Contract> => {
  const address = await Lib_AddressManager.getAddress(name)

  if (address === ZERO_ADDRESS) {
    throw new Error(
      `Lib_AddressManager does not have a record for a contract named: ${name}`
    )
  }

  return loadContract(name, address, provider)
}

export const loadProxyFromManager = async (
  name: string,
  proxy: string,
  Lib_AddressManager: Contract,
  provider: JsonRpcProvider
): Promise<Contract> => {
  const address = await Lib_AddressManager.getAddress(proxy)

  if (address === ZERO_ADDRESS) {
    throw new Error(
      `Lib_AddressManager does not have a record for a contract named: ${proxy}`
    )
  }

  return loadContract(name, address, provider)
}
