/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { deployAndRegister } from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  await deployAndRegister({
    hre,
    name: 'OVM_StateManagerFactory',
    args: [],
  })
}

deployFn.tags = ['OVM_FraudVerifier']

export default deployFn
