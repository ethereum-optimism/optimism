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
    name: 'OVM_ExecutionManager',
    args: [
      Lib_AddressManager.address,
      {
        minTransactionGasLimit: (hre as any).deployConfig
          .emMinTransactionGasLimit,
        maxTransactionGasLimit: (hre as any).deployConfig
          .emMaxTransactionGasLimit,
        maxGasPerQueuePerEpoch: (hre as any).deployConfig
          .emMaxGasPerQueuePerEpoch,
        secondsPerEpoch: (hre as any).deployConfig.emSecondsPerEpoch,
      },
      {
        ovmCHAINID: (hre as any).deployConfig.emOvmChainId,
      },
    ],
  })
}

deployFn.dependencies = ['Lib_AddressManager']
deployFn.tags = ['OVM_ExecutionManager']

export default deployFn
