/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getContractDefinition } from '../src'

const deployFn: DeployFunction = async (hre: any) => {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const gasPriceOracle = getContractDefinition('OVM_GasPriceOracle', true)

  await deploy('OVM_GasPriceOracle', {
    contract: gasPriceOracle,
    from: deployer,
    args: [(hre as any).deployConfig.ovmSequencerAddress],
    log: true,
  });
}

deployFn.tags = ['OVM_GasPriceOracle']

export default deployFn
