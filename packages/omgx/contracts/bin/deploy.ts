import { Wallet, providers } from 'ethers'
import { getContractFactory } from '@eth-optimism/contracts'

require('dotenv').config()

import hre from 'hardhat'

const main = async () => {

  console.log('Starting OMGX core contracts deployment...')

  //const config = parseEnv()
  //not clear if the output is used anywhere?

  const l1Provider = new providers.JsonRpcProvider(process.env.L1_NODE_WEB3_URL)
  const l2Provider = new providers.JsonRpcProvider(process.env.L2_NODE_WEB3_URL)

  const deployer_l1 = new Wallet(process.env.DEPLOYER_PRIVATE_KEY, l1Provider)
  const deployer_l2 = new Wallet(process.env.DEPLOYER_PRIVATE_KEY, l2Provider)

  const getAddressManager = (provider: any, addressManagerAddress: any) => {
    return getContractFactory('Lib_AddressManager')
      .connect(provider)
      .attach(addressManagerAddress) as any
  }

  console.log(`ADDRESS_MANAGER_ADDRESS was set to ${process.env.ADDRESS_MANAGER_ADDRESS}`)
  const addressManager = getAddressManager(deployer_l1, process.env.ADDRESS_MANAGER_ADDRESS);

  const l1MessengerAddress = await addressManager.getAddress(
    'Proxy__OVM_L1CrossDomainMessenger'
  )
  const l2MessengerAddress = await addressManager.getAddress(
    'OVM_L2CrossDomainMessenger'
  )

  const L1StandardBridgeAddress = await addressManager.getAddress(
    'Proxy__OVM_L1StandardBridge'
  )
  const L1StandardBridge = getContractFactory('OVM_L1StandardBridge')
    .connect(deployer_l1)
    .attach(L1StandardBridgeAddress)

  const L2StandardBridgeAddress = await L1StandardBridge.l2TokenBridge()

  await hre.run('deploy', {
    l1MessengerAddress,
    l2MessengerAddress,
    L1StandardBridgeAddress,
    L2StandardBridgeAddress,
    l1Provider,
    l2Provider,
    deployer_l1,
    deployer_l2,
    addressManager,
    noCompile: process.env.NO_COMPILE ? true : false,
  })

}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.log(
      JSON.stringify({ error: error.message, stack: error.stack }, null, 2)
    )
    process.exit(1)
  })

//Based on the code, does not seem to be used?
// function parseEnv() {

//   function ensure(env, type) {
//     if (typeof process.env[env] === 'undefined')
//       return undefined
//     if (type === 'number')
//       return parseInt(process.env[env], 10)
//     return process.env[env]
//   }

//   return {
//     l1provider: ensure('L1_NODE_WEB3_URL', 'string'),
//     l2provider: ensure('L2_NODE_WEB3_URL', 'string'),
//     deployer: ensure('DEPLOYER_PRIVATE_KEY', 'string'),
//     emOvmChainId: ensure('CHAIN_ID', 'number'),
//   }
// }
