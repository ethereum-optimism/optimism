/* External Imports */
import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { FakeContract } from '@defi-wonderland/smock'

export const setProxyTarget = async (
  AddressManager: Contract,
  name: string,
  target: FakeContract
): Promise<void> => {
  const SimpleProxy: Contract = await (
    await ethers.getContractFactory('Helper_SimpleProxy')
  ).deploy()

  await SimpleProxy.setTarget(target.address)
  await AddressManager.setAddress(name, SimpleProxy.address)
}
