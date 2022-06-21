/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { Contract } from 'ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  await deploy('L1CrossDomainMessenger', {
    from: deployer,
    args: [],
    log: true,
    waitConfirmations: 1,
  })

  const provider = hre.ethers.provider.getSigner(deployer)
  const portal = await hre.deployments.get('OptimismPortal')
  const messenger = await hre.deployments.get('L1CrossDomainMessenger')

  const L1CrossDomainMessenger = new Contract(
    messenger.address,
    messenger.abi,
    provider
  )

  const tx = await L1CrossDomainMessenger.initialize(portal.address)
  const receipt = await tx.wait()
  console.log(`${receipt.transactionHash}: initialize(${portal.address})`)
}

deployFn.tags = ['L1CrossDomainMessenger']

export default deployFn
