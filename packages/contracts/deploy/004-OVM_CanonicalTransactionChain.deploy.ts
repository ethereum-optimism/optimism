/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'
import { names } from '../src/address-names'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    names.unmanaged.Lib_AddressManager
  )

  await deployAndVerifyAndThen({
    hre,
    name: names.managed.contracts.CanonicalTransactionChain,
    args: [
      Lib_AddressManager.address,
      hre.deployConfig.l2BlockGasLimit,
      hre.deployConfig.ctcL2GasDiscountDivisor,
      hre.deployConfig.ctcEnqueueGasCost,
    ],
  })
}

deployFn.tags = ['CanonicalTransactionChain', 'upgrade']

export default deployFn
