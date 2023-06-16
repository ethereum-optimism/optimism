import { DeployFunction } from 'hardhat-deploy/dist/types'
import { awaitCondition } from '@eth-optimism/core-utils'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [proxyAdmin] = await getContractsFromArtifacts(hre, [
    {
      name: 'ProxyAdmin',
      signerOrProvider: deployer,
    },
  ])

  const finalOwner = hre.deployConfig.finalSystemOwner
  const proxyAdminOwner = await proxyAdmin.callStatic.owner()

  if (proxyAdminOwner !== finalOwner) {
    const tx = await proxyAdmin.transferOwnership(finalOwner)
    await tx.wait()

    await awaitCondition(
      async () => {
        return (await proxyAdmin.callStatic.owner()) === finalOwner
      },
      30000,
      1000
    )
  }
}

deployFn.tags = ['ProxyAdmin', 'transferOwnership', 'l1']

export default deployFn
