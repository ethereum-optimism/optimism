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
  ]

  const addresses = await Promise.all(
    names.map(async (n) => {
      return (await getDeployedContract(hre, n)).address
    })
  )

  // Add non-deployed addresses to the Address Setter argument arrays
  // L2CrossDomainMessenger is the address of the predeploy on L2. We can refactor off-chain
  // services such that we can remove the need to set this address, but for now it's easier
  // to simply keep setting the address.
  names.push('L2CrossDomainMessenger')
  addresses.push(predeploys.L2CrossDomainMessenger)

  // OVM_Sequencer is the address allowed to submit "Sequencer" blocks to the
  // CanonicalTransactionChain.
  names.push('OVM_Sequencer')
  addresses.push((hre as any).deployConfig.ovmSequencerAddress)

  // OVM_Proposer is the address allowed to submit state roots (transaction results) to the
  // StateCommitmentChain.
  names.push('OVM_Proposer')
  addresses.push((hre as any).deployConfig.ovmProposerAddress)

  await deployAndPostDeploy({
    hre,
    name: 'AddressSetter1',
    args: [
      Lib_AddressManager.address,
      (hre as any).deployConfig.ovmAddressManagerOwner,
      names,
      addresses,
    ],
  })
}

deployFn.tags = ['AddressSetter1', 'upgrade']

export default deployFn
