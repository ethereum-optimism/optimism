/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndPostDeploy,
  getDeployedContract,
  getReusableContract,
} from '../src/hardhat-deploy-ethers'
import { predeploys } from '../src/predeploys'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getReusableContract(
    hre,
    'Lib_AddressManager'
  )

  // ToDo: Clean up the method of mapping names to addresses esp.
  // There's probably a more functional way to generate an object or something.
  const names = [
    'ChainStorageContainer-CTC-batches',
    'ChainStorageContainer-SCC-batches',
    'CanonicalTransactionChain',
    'StateCommitmentChain',
    'BondManager',
    'OVM_L1CrossDomainMessenger',
    'Proxy__L1CrossDomainMessenger',
    'Proxy__L1StandardBridge',
  ]

  const addresses = await Promise.all(
    names.map(async (n) => {
      return (await getDeployedContract(hre, n)).address
    })
  )

  // Add non-deployed addresses to the arrays
  names.push('L2CrossDomainMessenger')
  addresses.push(predeploys.L2CrossDomainMessenger)
  names.push('OVM_Sequencer')
  addresses.push((hre as any).deployConfig.ovmSequencerAddress)
  names.push('OVM_Proposer')
  addresses.push((hre as any).deployConfig.ovmProposerAddress)

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
