/* External Imports */
import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { getContractFactory as ctFactory } from '@eth-optimism/contracts'

export const getContractFactory = async (contract: string) =>
  ctFactory(contract, (await ethers.getSigners())[0])

export const setProxyTarget = async (
  AddressManager: Contract,
  name: string,
  target: Contract
): Promise<void> => {
  const SimpleProxy: Contract = await (
    await getContractFactory('Helper_SimpleProxy')
  ).deploy()

  await SimpleProxy.setTarget(target.address)
  await AddressManager.setAddress(name, SimpleProxy.address)
}

export const makeAddressManager = async (): Promise<Contract> => {
  return (await getContractFactory('Lib_AddressManager')).deploy()
}
