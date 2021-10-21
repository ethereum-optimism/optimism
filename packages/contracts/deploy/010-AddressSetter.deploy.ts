/* Imports: External */
import { DeployFunction, DeploymentsExtension } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getDeployedContract,
  getReusableContract,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getReusableContract(hre, 'Lib_AddressManager')

  const names = [
    'ChainStorageContainer-CTC-batches',
    'ChainStorageContainer-SCC-batches',
    'CanonicalTransactionChain',
    'StateCommitmentChain',
    'BondManager',
    'OVM_L1CrossDomainMessenger',
    'Proxy__L1CrossDomainMessenger',
    'Proxy__L1StandardBridge',
    'OVM_Proposer',
  ]

  const addresses = await Promise.all(
    names.map(async (n) => {
      return (await getDeployedContract(hre, n)).address
    })
  )

  await deployAndPostDeploy({
    hre,
    name: 'AddressSetter',
    args: [
      Lib_AddressManager.address,
      (hre as any).deployConfig.ovmAddressManagerOwner,
      names,
      addresses,
    ],
  })
}

deployFn.tags = ['AddressSetter', 'upgrade']

export default deployFn
