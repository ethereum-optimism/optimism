/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract } from 'ethers'

const getLibraryConfig = (ProxyManager: Contract): any => {
  return [
    {
      name: 'Lib_ByteUtils',
      params: []
    },
    {
      name: 'Lib_EthUtils',
      params: [ProxyManager.address]
    },
    {
      name: 'Lib_RLPReader',
      params: []
    },
    {
      name: 'Lib_RLPWriter',
      params: []
    }
  ]
}

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

    await Proxy_Manager.setProxy(
      name,
      Proxy_Forwarder.address
    )
  }
}

export const setProxyTarget = async (
  Proxy_Manager: Contract,
  name: string,
  target: Contract
): Promise<void> => {
  await makeProxies(Proxy_Manager, [name])
  
  await Proxy_Manager.setTarget(
    name,
    target.address
  )
}

export const getProxyManager = async (): Promise<Contract> => {
  const Factory__Proxy_Manager = await ethers.getContractFactory(
    'Proxy_Manager'
  )

  const Proxy_Manager = await Factory__Proxy_Manager.deploy()

  const libraryConfig = getLibraryConfig(Proxy_Manager)

  await makeProxies(
    Proxy_Manager,
    libraryConfig.map((config) => {
      return config.name
    })
  )

  for (const config of libraryConfig) {
    const Factory__Lib_Contract = await ethers.getContractFactory(
      config.name
    )
    const Lib_Contract = await Factory__Lib_Contract.deploy(
      ...config.params
    )

    await Proxy_Manager.setTarget(
      config.name,
      Lib_Contract.address
    )
  }

  return Proxy_Manager
}
