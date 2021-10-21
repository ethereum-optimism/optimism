/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getDeployedContract,
  getLibAddressManager,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getLibAddressManager(hre)

  await deployAndPostDeploy({
    hre,
    name: 'BondManager',
    args: [Lib_AddressManager.address],
  })
}

deployFn.tags = ['BondManager', 'upgrade']

export default deployFn
