/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getDeployedContract } from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  const Lib_AddressManager = await getDeployedContract(
    hre,
    'Lib_AddressManager',
    {
      signerOrProvider: deployer,
    }
  )

  const result = await deploy('Proxy__L1CrossDomainMessenger', {
    contract: 'Lib_ResolvedDelegateProxy',
    from: deployer,
    args: [Lib_AddressManager.address, 'L1CrossDomainMessenger'],
    log: true,
  })

  if (!result.newlyDeployed) {
    return
  }

  const Proxy__L1CrossDomainMessenger = await getDeployedContract(
    hre,
    'Proxy__L1CrossDomainMessenger',
    {
      signerOrProvider: deployer,
      iface: 'L1CrossDomainMessenger',
    }
  )

  const tx = await Proxy__L1CrossDomainMessenger.initialize(
    Lib_AddressManager.address
  )
  await tx.wait()

  const libAddressManager =
    await Proxy__L1CrossDomainMessenger.libAddressManager()
  if (libAddressManager !== Lib_AddressManager.address) {
    throw new Error(
      `\n**FATAL ERROR. THIS SHOULD NEVER HAPPEN. CHECK YOUR DEPLOYMENT.**:\n` +
        `Proxy__L1CrossDomainMessenger could not be succesfully initialized.\n` +
        `Attempted to set Lib_AddressManager to: ${Lib_AddressManager.address}\n` +
        `Actual address after initialization: ${libAddressManager}\n` +
        `This could indicate a compromised deployment.`
    )
  }

  await Lib_AddressManager.setAddress(
    'Proxy__L1CrossDomainMessenger',
    result.address
  )
}

deployFn.dependencies = ['Lib_AddressManager', 'L1CrossDomainMessenger']
deployFn.tags = ['Proxy__L1CrossDomainMessenger']

export default deployFn
