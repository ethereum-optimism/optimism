/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getContractDefinition } from '../src'

const deployFn: DeployFunction = async (hre: any) => {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const gasPriceOracle = getContractDefinition('OVM_GasPriceOracle', true)

  const gasOracleOwner = (hre as any).deployConfig.ovmSequencerAddress
  const initialGasPrice = (hre as any).deployConfig.initialGasPriceOracleGasPrice

  if (!gasOracleOwner || !initialGasPrice) {
    throw new Error('initialGasPrice & ovmSequencerAddress required to deploy gas price oracle')
  }

  await deploy('OVM_GasPriceOracle', {
    contract: gasPriceOracle,
    from: deployer,
    args: [gasOracleOwner, initialGasPrice],
    log: true,
  });
}

deployFn.tags = ['OVM_GasPriceOracle']

export default deployFn
