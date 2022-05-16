/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { Contract } from 'ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  await deploy('OptimismMintableTokenFactory', {
    from: deployer,
    args: [],
    log: true,
    waitConfirmations: 1,
  })

  const provider = hre.ethers.provider.getSigner(deployer)

  const factory = await hre.deployments.get('OptimismMintableTokenFactory')
  const bridge = await hre.deployments.get('L1StandardBridge')

  const OptimismMintableTokenFactory = new Contract(
    factory.address,
    factory.abi,
    provider
  )

  const tx = await OptimismMintableTokenFactory.initialize(bridge.address)
  const receipt = await tx.wait()
  console.log(`${receipt.transactionHash}: initialize(${bridge.address})`)
}

deployFn.tags = ['OptimismMintableTokenFactory']

export default deployFn
