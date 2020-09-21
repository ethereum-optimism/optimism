/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract } from 'ethers'

export const makeProxies = async (
  Proxy_Manager: Contract,
  names: string[]
): Promise<void> => {
  for (const name of names) {
    if (await Proxy_Manager['hasProxy(string)'](name)) {
      continue
    }

    const Factory__Proxy_Forwarder = await ethers.getContractFactory(
      'Proxy_Forwarder'
    )

    const Proxy_Forwarder = await Factory__Proxy_Forwarder.deploy(
      Proxy_Manager.address
    )

    await Proxy_Manager.setProxy(name, Proxy_Forwarder.address)
  }
}

export const setProxyTarget = async (
  Proxy_Manager: Contract,
  name: string,
  target: Contract
): Promise<void> => {
  await makeProxies(Proxy_Manager, [name])

  await Proxy_Manager.setTarget(name, target.address)
}

export const getProxyManager = async (): Promise<Contract> => {
  return (await ethers.getContractFactory('Proxy_Manager')).deploy()
}
