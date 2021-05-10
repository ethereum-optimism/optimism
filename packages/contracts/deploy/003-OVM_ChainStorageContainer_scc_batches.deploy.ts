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
    name: 'OVM_ChainStorageContainer-SCC-batches',
    contract: 'OVM_ChainStorageContainer',
    args: [Lib_AddressManager.address, 'OVM_StateCommitmentChain'],
  })
}

deployFn.dependencies = ['Lib_AddressManager']
deployFn.tags = ['OVM_ChainStorageContainer_scc_batches']

export default deployFn
