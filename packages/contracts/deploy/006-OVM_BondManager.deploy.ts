/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getContractFromArtifact,
} from '../src/hardhat-deploy-ethers'
import { addressNames } from '../src'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    addressNames.addressManager
  )

  await deployAndPostDeploy({
    hre,
    name: 'BondManager',
    args: [Lib_AddressManager.address],
  })
}

deployFn.tags = ['BondManager', 'upgrade']

export default deployFn
