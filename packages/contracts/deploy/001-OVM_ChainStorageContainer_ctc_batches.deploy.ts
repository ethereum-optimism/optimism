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
    name: 'ChainStorageContainer-CTC-batches',
    contract: 'ChainStorageContainer',
    args: [Lib_AddressManager.address, 'CanonicalTransactionChain'],
  })
}

deployFn.tags = ['upgrade', 'ChainStorageContainer_ctc_batches']

export default deployFn
