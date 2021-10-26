/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getContractFromArtifact,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    'Lib_AddressManager'
  )

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

deployFn.tags = ['upgrade', 'CanonicalTransactionChain']

export default deployFn
