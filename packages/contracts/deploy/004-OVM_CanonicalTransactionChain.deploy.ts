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
  const MVM_AddressManager = await getDeployedContract(
    hre,
    'MVM_AddressManager'
  )

  await deployAndRegister({
    hre,
    name: 'OVM_CanonicalTransactionChain',
    args: [
      MVM_AddressManager.address,
      Lib_AddressManager.address,
      (hre as any).deployConfig.ctcForceInclusionPeriodSeconds,
      (hre as any).deployConfig.ctcForceInclusionPeriodBlocks,
      (hre as any).deployConfig.ctcMaxTransactionGasLimit,
    ],
  })
}

deployFn.dependencies = ['Lib_AddressManager','MVM_AddressManager']
deployFn.tags = ['OVM_CanonicalTransactionChain']

export default deployFn
