/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    'Lib_AddressManager'
  )

  await deployAndVerifyAndThen({
    hre,
    name: 'ChainStorageContainer-SCC-batches',
    contract: 'ChainStorageContainer',
    args: [Lib_AddressManager.address, 'StateCommitmentChain'],
  })
}

deployFn.tags = ['ChainStorageContainer_scc_batches', 'upgrade']

export default deployFn
