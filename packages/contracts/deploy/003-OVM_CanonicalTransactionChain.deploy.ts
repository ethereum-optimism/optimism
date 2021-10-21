/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getDeployedContract,
  getReusableContract,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getReusableContract(hre, 'Lib_AddressManager')

  await deployAndPostDeploy({
    hre,
    name: 'CanonicalTransactionChain',
    args: [
      Lib_AddressManager.address,
      (hre as any).deployConfig.ctcMaxTransactionGasLimit,
      (hre as any).deployConfig.ctcL2GasDiscountDivisor,
      (hre as any).deployConfig.ctcEnqueueGasCost,
    ],
  })
}

deployFn.tags = ['CanonicalTransactionChain', 'upgrade']

export default deployFn
