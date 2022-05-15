/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { Contract } from 'ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  await deploy('L1StandardBridge', {
    from: deployer,
    args: [],
    log: true,
    waitConfirmations: 1,
  })

  const provider = hre.ethers.provider.getSigner(deployer)

  const messenger = await hre.deployments.get('L1CrossDomainMessenger')
  const bridge = await hre.deployments.get('L1StandardBridge')

  const L1StandardBridge = new Contract(bridge.address, bridge.abi, provider)

  const tx = await L1StandardBridge.initialize(messenger.address)
  const receipt = await tx.wait()
  console.log(`${receipt.transactionHash}: initialize(${messenger.address})`)
}

deployFn.tags = ['L1StandardBridge']

export default deployFn
