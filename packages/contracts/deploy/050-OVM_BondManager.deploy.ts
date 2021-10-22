/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getDeployedContract,
  getReusableContract,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getReusableContract(
    hre,
    'Lib_AddressManager'
  )

  await deployAndPostDeploy({
    hre,
    name: 'BondManager',
    args: [Lib_AddressManager.address],
  })
}

deployFn.tags = ['fresh', 'upgrade', 'BondManager']

export default deployFn
