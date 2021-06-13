/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getDeployedContract } from '../src/hardhat-deploy-ethers'
import { predeploys } from '../src/predeploys'

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

  const result = await deploy('Proxy__MVM_L1MetisGateway', {
    contract: 'Lib_ResolvedDelegateProxy',
    from: deployer,
    args: [Lib_AddressManager.address, 'MVM_L1MetisGateway'],
    log: true,
  })

  if (!result.newlyDeployed) {
    return
  }

  const Proxy__MVM_L1MetisGateway = await getDeployedContract(
    hre,
    'Proxy__MVM_L1MetisGateway',
    {
      signerOrProvider: deployer,
      iface: 'OVM_L1ERC20Gateway',
    }
  )

  await Proxy__MVM_L1MetisGateway.initialize(
    Lib_AddressManager.address,
    predeploys.MVM_Coinbase,
    '0xe552Fb52a4F19e44ef5A967632DBc320B0820639'   //TODO: rinkeby change mainnet
         
  )

  const libAddressManager = await Proxy__MVM_L1MetisGateway.libAddressManager()
  if (libAddressManager !== Lib_AddressManager.address) {
    throw new Error(
      `\n**FATAL ERROR. THIS SHOULD NEVER HAPPEN. CHECK YOUR DEPLOYMENT.**:\n` +
        `Proxy__OVM_L1ETHGateway could not be succesfully initialized.\n` +
        `Attempted to set Lib_AddressManager to: ${Lib_AddressManager.address}\n` +
        `Actual address after initialization: ${libAddressManager}\n` +
        `This could indicate a compromised deployment.`
    )
  }

  await Lib_AddressManager.setAddress('Proxy__MVM_L1MetisGateway', result.address)
}

deployFn.dependencies = ['Lib_AddressManager', 'MVM_L1MetisGateway']
deployFn.tags = ['Proxy__MVM_L1MetisGateway']

export default deployFn
