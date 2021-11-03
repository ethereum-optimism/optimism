/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getContractFromArtifact,
} from '../src/hardhat-deploy-ethers'
import { addressNames } from '../src'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    addressNames.addressManager
  )

  await deployAndPostDeploy({
    hre,
    name: 'ChainStorageContainer-SCC-batches',
    contract: 'ChainStorageContainer',
    args: [Lib_AddressManager.address, 'StateCommitmentChain'],
  })
}

deployFn.tags = ['ChainStorageContainer_scc_batches', 'upgrade']

export default deployFn
