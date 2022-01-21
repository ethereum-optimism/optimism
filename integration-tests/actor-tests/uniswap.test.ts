import { Contract, utils, Wallet } from 'ethers'
import { FeeAmount } from '@uniswap/v3-sdk'
import { abi as NFTABI } from '@uniswap/v3-periphery/artifacts/contracts/NonfungiblePositionManager.sol/NonfungiblePositionManager.json'
import { abi as RouterABI } from '@uniswap/v3-periphery/artifacts/contracts/SwapRouter.sol/SwapRouter.json'

import { actor, run, setupActor, setupRun } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'
import ERC20 from '../artifacts/contracts/ERC20.sol/ERC20.json'

interface Context {
  contracts: { [name: string]: Contract }
  wallet: Wallet
}

actor('Uniswap swapper', () => {
  let env: OptimismEnv

  let tokens: [Contract, Contract]

  let contracts: { [name: string]: Contract }

  setupActor(async () => {
    env = await OptimismEnv.new()

    contracts = {
      positionManager: new Contract(
        process.env.UNISWAP_POSITION_MANAGER_ADDRESS,
        NFTABI
      ).connect(env.l2Wallet),
      router: new Contract(
        process.env.UNISWAP_ROUTER_ADDRESS,
        RouterABI
      ).connect(env.l2Wallet),
    }

    tokens = [
      new Contract(process.env.UNISWAP_TOKEN_0_ADDRESS, ERC20.abi).connect(
        env.l2Wallet
      ),
      new Contract(process.env.UNISWAP_TOKEN_1_ADDRESS, ERC20.abi).connect(
        env.l2Wallet
      ),
    ]
    tokens =
      tokens[0].address.toLowerCase() < tokens[1].address.toLowerCase()
        ? [tokens[0], tokens[1]]
        : [tokens[1], tokens[0]]
  })

  setupRun(async () => {
    const wallet = Wallet.createRandom().connect(env.l2Provider)

    await env.l2Wallet.sendTransaction({
      to: wallet.address,
      value: utils.parseEther('0.1'),
    })

    for (const token of tokens) {
      let tx = await token.transfer(wallet.address, 1000000)
      await tx.wait()
      const boundToken = token.connect(wallet)
      tx = await boundToken.approve(
        contracts.positionManager.address,
        1000000000
      )
      await tx.wait()
      tx = await boundToken.approve(contracts.router.address, 1000000000)
      await tx.wait()
    }

    return {
      contracts: Object.entries(contracts).reduce((acc, [name, value]) => {
        acc[name] = value.connect(wallet)
        return acc
      }, {}),
      wallet,
    }
  })

  run(async (b, ctx: Context) => {
    await b.bench('swap', async () => {
      const tx = await ctx.contracts.router.exactInputSingle(
        {
          tokenIn: tokens[0].address,
          tokenOut: tokens[1].address,
          fee: FeeAmount.MEDIUM,
          recipient: ctx.wallet.address,
          deadline: Date.now() * 2,
          amountIn: Math.max(Math.floor(1000 * Math.random()), 100),
          amountOutMinimum: 0,
          sqrtPriceLimitX96: 0,
        },
        {
          gasLimit: 10000000,
        }
      )
      await tx.wait()
    })
  })
})
