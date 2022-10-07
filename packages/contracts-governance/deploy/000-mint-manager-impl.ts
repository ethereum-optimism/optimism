import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import 'hardhat-deploy'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import { sleep } from '@eth-optimism/core-utils'

// $ cast call --rpc-url https://mainnet.optimism.io 0x4200000000000000000000000000000000000042 'owner()(address)'
// 0x724604DB3C8D86c906A27B610703fD0296Eb26D5
// $ cast call --rpc-url https://mainnet.optimism.io 0x724604DB3C8D86c906A27B610703fD0296Eb26D5 'owner()(address)'
// 0x2A82Ae142b2e62Cb7D10b55E323ACB1Cab663a26
//
// $ cast call --rpc-url https://goerli.optimism.io 0x4200000000000000000000000000000000000042 'owner()(address)'
// 0xB2DcB2Df7732030eD99241E5F9aEBB3Fb53eFCb6
// $ cast call --rpc-url https://goerli.optimism.io 0xB2DcB2Df7732030eD99241E5F9aEBB3Fb53eFCb6  'owner()(address)'
// 0xC30276833798867C1dBC5c468bf51cA900b44E4c

// Deployment steps:
//  1 - Deploy new MintManager implementation
//  2 - Multisig transaction calls `upgrade()` on the old MintManager
const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  const upgrader = hre.deployConfig.upgrader
  const governanceToken = '0x4200000000000000000000000000000000000042'

  if (upgrader === '' || upgrader === hre.ethers.constants.AddressZero) {
    throw new Error('upgrader not set in deploy-config')
  }

  // There is no artifact for the originally deployed MintManager
  let oldImpl: string
  if (hre.network.name === 'optimism-mainnet') {
    oldImpl = '0x724604DB3C8D86c906A27B610703fD0296Eb26D5'
  } else if (hre.network.name === 'optimism-goerli') {
    oldImpl = '0xB2DcB2Df7732030eD99241E5F9aEBB3Fb53eFCb6'
  } else {
    throw new Error(`unknown network ${hre.network.name}`)
  }

  await deploy('MintManager', {
    from: deployer,
    args: [upgrader, governanceToken],
    log: true,
    waitConfirmations: 1,
  })

  const GovernanceToken = await hre.ethers.getContractAt(
    'GovernanceToken',
    '0x4200000000000000000000000000000000000042'
  )

  const Old__MintManager = await hre.ethers.getContractAt(
    'MintManager',
    oldImpl
  )
  const oldOwner = await Old__MintManager.owner()

  const getAddress = hre.ethers.utils.getAddress

  const Deployment__MintManager = await hre.deployments.get('MintManager')
  const newImpl = Deployment__MintManager.address

  console.log()
  console.log('Action is required to complete this deployment')
  console.log(
    'Ownership of the GovernanceToken should be migrated to the newly deployed MintManager'
  )
  console.log(`MintManager.owner() -> ${oldOwner}`)
  console.log(`Call MintManager(${oldImpl}).upgrade(${newImpl})`)
  console.log()

  while (true) {
    const owner = await GovernanceToken.owner()
    if (getAddress(owner) === getAddress(oldImpl)) {
      console.log(`GovernanceToken.owner() is still old implementation`)
    } else if (getAddress(owner) === getAddress(newImpl)) {
      console.log(`GovernanceToken.owner() upgraded to new implementation!`)
      break
    } else {
      throw new Error(
        `GovernanceToken.owner() upgraded to unknown address ${owner}`
      )
    }
    await sleep(5000)
  }
}

deployFn.tags = ['MintManager']

export default deployFn
