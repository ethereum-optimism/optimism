import { task, types } from 'hardhat/config'
import { providers } from 'ethers'
import '@nomiclabs/hardhat-ethers'

import { predeploys, getContractInterface } from '../src'

task('check-l2-config', 'Validate L2 config')
  .addParam(
    'l2ProviderUrl',
    'L2 provider URL.',
    'http://localhost:9545',
    types.string
  )
  .setAction(async (args, hre) => {
    const { l2ProviderUrl } = args
    const l2Provider = new providers.JsonRpcProvider(l2ProviderUrl)

    const OptimismMintableERC20Factory = new hre.ethers.Contract(
      predeploys.OptimismMintableERC20Factory,
      getContractInterface('OptimismMintableERC20Factory'),
      l2Provider
    )

    const bridge = await OptimismMintableERC20Factory.bridge()
    console.log(`OptimismMintableERC20Factory.bridge() -> ${bridge}`)
    if (bridge !== predeploys.L2StandardBridge) {
      throw new Error(
        `L2StandardBridge not set correctly. Got ${bridge}, expected ${predeploys.L2StandardBridge}`
      )
    }
  })
