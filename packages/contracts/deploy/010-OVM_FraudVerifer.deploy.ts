/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndRegister,
  getDeployedContract,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getDeployedContract(
    hre,
    'Lib_AddressManager'
  )

  await deployAndRegister({
    hre,
    name: 'OVM_FraudVerifier',
    args: [Lib_AddressManager.address],
  })
}

deployFn.dependencies = ['Lib_AddressManager']
deployFn.tags = ['OVM_FraudVerifier']

export default deployFn
