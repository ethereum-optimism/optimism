import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [proxyAdmin, mintableERC20FactoryProxy, mintableERC20FactoryImpl] =
    await getContractsFromArtifacts(hre, [
      {
        name: 'ProxyAdmin',
        signerOrProvider: deployer,
      },
      {
        name: 'OptimismMintableERC20FactoryProxy',
        iface: 'OptimismMintableERC20Factory',
        signerOrProvider: deployer,
      },
      {
        name: 'OptimismMintableERC20Factory',
      },
    ])

  try {
    const tx = await proxyAdmin.upgrade(
      mintableERC20FactoryProxy.address,
      mintableERC20FactoryImpl.address
    )
    await tx.wait()
  } catch (e) {
    console.log('OptimismMintableERC20Factory already initialized')
  }

  const version = await mintableERC20FactoryProxy.callStatic.version()
  console.log(`OptimismMintableERC20Factory version: ${version}`)

  console.log('Upgraded OptimismMintableERC20Factory')
}

deployFn.tags = ['OptimismMintableERC20FactoryInitialize', 'l1']

export default deployFn
