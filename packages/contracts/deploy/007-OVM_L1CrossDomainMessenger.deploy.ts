/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  getDeployedContract,
  deployAndRegister,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManager = await getDeployedContract(
    hre,
    'Lib_AddressManager'
  )

  await deployAndRegister({
    hre,
    name: 'L1CrossDomainMessenger',
    args: [],
    postDeployAction: async (contract) => {
      // Theoretically it's not necessary to initialize this contract since it sits behind
      // a proxy. However, it's best practice to initialize it anyway just in case there's
      // some unknown security hole. It also prevents another user from appearing like an
      // official address because it managed to call the initialization function.
      console.log(`Initializing L1CrossDomainMessenger...`)
      await contract.initialize(Lib_AddressManager.address)

      console.log(`Checking that contract was correctly initialized...`)
      await waitUntilTrue(async () => {
        return hexStringEquals(
          await contract.libAddressManager(),
          Lib_AddressManager.address
        )
      })
    },
  })
}

deployFn.dependencies = ['Lib_AddressManager']
deployFn.tags = ['L1CrossDomainMessenger']

export default deployFn
