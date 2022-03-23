/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'
import { getDeployConfig } from '../src/deploy-config'
import { names } from '../src/address-names'

const deployFn: DeployFunction = async (hre) => {
  const deployConfig = getDeployConfig(hre.network.name)

  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    names.unmanaged.Lib_AddressManager
  )

  await deployAndVerifyAndThen({
    hre,
    name: names.managed.contracts.CanonicalTransactionChain,
    args: [
      Lib_AddressManager.address,
      deployConfig.l2BlockGasLimit,
      deployConfig.ctcL2GasDiscountDivisor,
      deployConfig.ctcEnqueueGasCost,
    ],
  })
}

deployFn.tags = ['CanonicalTransactionChain', 'upgrade']

export default deployFn
