import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  // The DisputeGameFactory is only deployed on devnet
  if (hre.deployConfig.l1ChainID === 900) {
    const { deployer } = await hre.getNamedAccounts()
    const [proxyAdmin, disputeGameFactoryProxy, disputeGameFactoryImpl] =
      await getContractsFromArtifacts(hre, [
        {
          name: 'ProxyAdmin',
          signerOrProvider: deployer,
        },
        {
          name: 'DisputeGameFactoryProxy',
          iface: 'DisputeGameFactory',
          signerOrProvider: deployer,
        },
        {
          name: 'DisputeGameFactory',
        },
      ])

    const finalOwner = hre.deployConfig.finalSystemOwner

    try {
      const tx = await proxyAdmin.upgradeAndCall(
        disputeGameFactoryProxy.address,
        disputeGameFactoryImpl.address,
        disputeGameFactoryProxy.interface.encodeFunctionData('initialize', [
          finalOwner,
        ])
      )
      await tx.wait()
    } catch (e) {
      console.log('DisputeGameFactory already initialized')
    }

    const fetchedOwner = await disputeGameFactoryProxy.callStatic.owner()
    assert(fetchedOwner === finalOwner)

    console.log('Updgraded and initialized DisputeGameFactory')
  }
}

deployFn.tags = ['DisputeGameFactoryInitialize', 'l1']

export default deployFn
