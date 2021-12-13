import { BigNumber, Contract, utils, Wallet, ContractFactory } from 'ethers'
import { actor, run, setupActor, setupRun } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'
import { UniswapV3Deployer } from 'uniswap-v3-deploy-plugin/dist/deployer/UniswapV3Deployer'
import { FeeAmount, TICK_SPACINGS } from '@uniswap/v3-sdk'
import ERC20 from '../artifacts/contracts/ERC20.sol/ERC20.json'

interface Context {
  contracts: { [name: string]: Contract }
  wallet: Wallet
}

// Below methods taken from the Uniswap test suite, see
// https://github.com/Uniswap/v3-periphery/blob/main/test/shared/ticks.ts
export const getMinTick = (tickSpacing: number) =>
  Math.ceil(-887272 / tickSpacing) * tickSpacing
export const getMaxTick = (tickSpacing: number) =>
  Math.floor(887272 / tickSpacing) * tickSpacing

actor('Uniswap swapper', () => {
  let env: OptimismEnv

  let tokens: [Contract, Contract]

  let contracts: { [name: string]: Contract }

  setupActor(async () => {
    env = await OptimismEnv.new()

    const factory = new ContractFactory(ERC20.abi, ERC20.bytecode, env.l2Wallet)
    const tokenA = await factory.deploy(1000000000, 'OVM1', 8, 'OVM1')
    await tokenA.deployed()
    const tokenB = await factory.deploy(1000000000, 'OVM2', 8, 'OVM2')
    await tokenB.deployed()

    tokens =
      tokenA.address < tokenB.address ? [tokenA, tokenB] : [tokenB, tokenA]
    contracts = await UniswapV3Deployer.deploy(env.l2Wallet)

    let tx
    for (const token of tokens) {
      tx = await token.approve(contracts.positionManager.address, 1000000000)
      await tx.wait()
      tx = await token.approve(contracts.router.address, 1000000000)
      await tx.wait()
    }

    tx = await contracts.positionManager.createAndInitializePoolIfNecessary(
      tokens[0].address,
      tokens[1].address,
      FeeAmount.MEDIUM,
      // initial ratio of 1/1
      BigNumber.from('79228162514264337593543950336')
    )
    await tx.wait()

    tx = await contracts.positionManager.mint(
      {
        token0: tokens[0].address,
        token1: tokens[1].address,
        tickLower: getMinTick(TICK_SPACINGS[FeeAmount.MEDIUM]),
        tickUpper: getMaxTick(TICK_SPACINGS[FeeAmount.MEDIUM]),
        fee: FeeAmount.MEDIUM,
        recipient: env.l2Wallet.address,
        amount0Desired: 100000000,
        amount1Desired: 100000000,
        amount0Min: 0,
        amount1Min: 0,
        deadline: Date.now() * 2,
      },
      {
        gasLimit: 10000000,
      }
    )
    await tx.wait()
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
