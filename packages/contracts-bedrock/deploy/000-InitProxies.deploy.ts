import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const proxies = [
  'L2OutputOracleProxy',
  'L1CrossDomainMessengerProxy',
  'L1StandardBridgeProxy',
  'OptimismPortalProxy',
  'OptimismMintableERC20FactoryProxy',
  'L1ERC721BridgeProxy',
]

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre
  const l1 = hre.ethers.provider

  const promises = []
  const nonce = await l1.getTransactionCount(deployer)
  for (let i = 0; i < proxies.length; i++) {
    const proxy = proxies[i]
    promises.push(
      deploy(proxy, {
        contract: 'Proxy',
        from: deployer,
        args: [deployer],
        log: true,
        waitConfirmations: deployConfig.deploymentWaitConfirmations,
        nonce: nonce + i,
      })
    )
  }

  await Promise.all(promises)
}

deployFn.tags = ['InitProxies']

export default deployFn
