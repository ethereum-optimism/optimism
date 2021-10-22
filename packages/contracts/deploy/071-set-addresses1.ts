/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getDeployedContract } from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  // todo: add waitUntilTrue, detect when AddressSetter1 has ownership of the AddressManager
  await (await getDeployedContract(hre, 'AddressSetter1')).setAddresses()
}

deployFn.tags = ['set-addresses', 'upgrade']

export default deployFn
