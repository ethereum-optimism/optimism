/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getContractDefinition } from '../src'

const deployFn: DeployFunction = async (hre: any) => {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const l2TokenFactory = getContractDefinition('OVM_L2StandardTokenFactory', true)

  await deploy('OVM_L2StandardTokenFactory', {
    contract: l2TokenFactory,
    args: [],
    from: deployer,
    log: true,
  });
}

deployFn.tags = ['OVM_L2StandardTokenFactory']

export default deployFn
