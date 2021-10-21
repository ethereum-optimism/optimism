/* eslint @typescript-eslint/no-var-requires: "off" */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import {
  getAdvancedContract,
  waitUntilTrue,
} from '../src/hardhat-deploy-ethers'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  if ((hre as any).deployConfig.forked !== 'true') {
    return
  }

  console.log(`Running custom setup for forked experimental networks`)

  // Fund the deployer account
  console.log(`Funding deployer account...`)
  const amount = `0xFFFFFFFFFFFFFFFFFF`
  await hre.ethers.provider.send('hardhat_setBalance', [deployer, amount])

  console.log(`Waiting for balance to reflect...`)
  await waitUntilTrue(async () => {
    const balance = await hre.ethers.provider.getBalance(deployer)
    return balance.gte(hre.ethers.BigNumber.from(amount))
  })

  // Get a reference to the AddressManager contract.
  const artifact = await hre.deployments.get('Lib_AddressManager')
  const Lib_AddressManager = getAdvancedContract({
    hre,
    contract: new hre.ethers.Contract(
      artifact.address,
      artifact.abi,
      hre.ethers.provider
    ),
  })

  // Impersonate the owner of the AddressManager
  console.log(`Impersonating owner account...`)
  const owner = await Lib_AddressManager.owner()
  await hre.ethers.provider.send('hardhat_impersonateAccount', [owner])

  console.log(`Started impersonating ${owner}`)
  console.log(`Setting AddressManager owner to ${deployer}`)
  const signer = await hre.ethers.getSigner(owner)
  await Lib_AddressManager.connect(signer).transferOwnership(deployer)

  console.log(`Waiting for owner to be correctly set...`)
  await waitUntilTrue(async () => {
    return (await Lib_AddressManager.owner()) === deployer
  })

  console.log(`Disabling impersonation...`)
  await hre.ethers.provider.send('hardhat_stopImpersonatingAccount', [owner])
}

deployFn.tags = ['hardhat', 'upgrade']

export default deployFn
