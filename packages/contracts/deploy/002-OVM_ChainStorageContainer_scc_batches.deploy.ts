/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getLiveContract,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getLiveContract(hre, 'Lib_AddressManager')

  await deployAndPostDeploy({
    hre,
    name: 'ChainStorageContainer-SCC-batches',
    contract: 'ChainStorageContainer',
    args: [Lib_AddressManager.address, 'StateCommitmentChain'],
  })
}

deployFn.tags = ['upgrade', 'ChainStorageContainer_scc_batches']

export default deployFn
