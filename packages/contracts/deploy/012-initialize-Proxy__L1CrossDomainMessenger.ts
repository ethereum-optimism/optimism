/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals, awaitCondition } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { getContractFromArtifact } from '../src/deploy-utils'
import { names } from '../src/address-names'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  // There's a risk that we could get front-run during a fresh deployment, which would brick this
  // contract and require that the proxy be re-deployed. We will not have this risk once we move
  // entirely to chugsplash-style deployments. It's unlikely to happen and relatively easy to
  // recover from so let's just ignore it for now.
  const Proxy__OVM_L1CrossDomainMessenger = await getContractFromArtifact(
    hre,
    names.managed.contracts.Proxy__OVM_L1CrossDomainMessenger,
    {
      iface: 'L1CrossDomainMessenger',
      signerOrProvider: deployer,
    }
  )

  const Lib_AddressManager = await getContractFromArtifact(
    hre,
    names.unmanaged.Lib_AddressManager
  )

  console.log(`Initializing Proxy__OVM_L1CrossDomainMessenger...`)
  await Proxy__OVM_L1CrossDomainMessenger.initialize(Lib_AddressManager.address)

  console.log(`Checking that contract was correctly initialized...`)
  await awaitCondition(
    async () => {
      return hexStringEquals(
        await Proxy__OVM_L1CrossDomainMessenger.libAddressManager(),
        Lib_AddressManager.address
      )
    },
    5000,
    100
  )

  console.log(`Setting Proxy__OVM_L1CrossDomainMessenger owner...`)
  const owner = hre.deployConfig.ovmAddressManagerOwner
  await Proxy__OVM_L1CrossDomainMessenger.transferOwnership(owner)

  console.log(`Checking that the contract owner was correctly set...`)
  await awaitCondition(
    async () => {
      return hexStringEquals(
        await Proxy__OVM_L1CrossDomainMessenger.owner(),
        owner
      )
    },
    5000,
    100
  )
}

deployFn.tags = ['finalize']

export default deployFn
