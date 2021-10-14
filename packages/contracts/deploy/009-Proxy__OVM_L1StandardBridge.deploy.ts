/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { deployAndRegister } from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  await deployAndRegister({
    hre,
    name: 'Proxy__L1StandardBridge',
    contract: 'L1ChugSplashProxy',
    iface: 'L1StandardBridge',
    args: [deployer],
  })
}

deployFn.tags = ['Proxy__L1StandardBridge']

export default deployFn
