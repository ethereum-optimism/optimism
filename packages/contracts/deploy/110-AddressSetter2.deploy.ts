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

  const names = ['Proxy__L1CrossDomainMessenger', 'Proxy__L1StandardBridge']

  const addresses = await Promise.all(
    names.map(async (n) => {
      return (await getDeployedContract(hre, n)).address
    })
  )

  await deployAndPostDeploy({
    hre,
    name: 'AddressSetter2',
    args: [
      Lib_AddressManager.address,
      (hre as any).deployConfig.ovmAddressManagerOwner,
      names,
      addresses,
    ],
  })
}

deployFn.tags = ['fresh', 'upgrade', 'AddressSetter2']

export default deployFn
