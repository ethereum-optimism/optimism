/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { deployAndPostDeploy } from '../src/hardhat-deploy-ethers'
import { addressNames } from '../src'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  await deployAndPostDeploy({
    hre,
    name: 'Proxy__OVM_L1StandardBridge',
    contract: 'L1ChugSplashProxy',
    iface: 'L1StandardBridge',
    args: [deployer],
  })
}

// This is kept during an upgrade. So no upgrade tag.
deployFn.tags = ['Proxy__OVM_L1StandardBridge']

export default deployFn
