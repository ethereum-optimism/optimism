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
    name: names.managed.contracts.ChainStorageContainer_SCC_batches,
    contract: 'ChainStorageContainer',
    args: [Lib_AddressManager.address, 'StateCommitmentChain'],
  })
}

deployFn.tags = ['ChainStorageContainer_scc_batches', 'upgrade']

export default deployFn
