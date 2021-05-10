/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import {
  deployAndRegister,
  getDeployedContract,
  registerAddress,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getDeployedContract(
    hre,
    'Lib_AddressManager'
  )

  await deployAndRegister({
    hre,
    name: 'OVM_L1MultiMessageRelayer',
    args: [Lib_AddressManager.address],
  })

  // OVM_L2MessageRelayer *must* be set to multi message relayer address on mainnet.
  if (hre.network.name.includes('mainnet')) {
    const OVM_L1MultiMessageRelayer = await getDeployedContract(
      hre,
      'OVM_L1MultiMessageRelayer'
    )

    await registerAddress({
      hre,
      name: 'OVM_L2MessageRelayer',
      address: OVM_L1MultiMessageRelayer.address
    })
  }
}

deployFn.dependencies = ['Lib_AddressManager']
deployFn.tags = ['OVM_L1MultiMessageRelayer']

export default deployFn
