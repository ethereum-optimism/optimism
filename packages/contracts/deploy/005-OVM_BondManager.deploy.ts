/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getContractFromArtifact,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    'Lib_AddressManager'
  )

  await deployAndPostDeploy({
    hre,
    name: 'BondManager',
    args: [Lib_AddressManager.address],
  })
}

deployFn.tags = ['upgrade', 'BondManager']

export default deployFn
