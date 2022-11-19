/* Imports: External */
import { BigNumber, Contract, ContractFactory, utils, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import { UniswapV3Deployer } from 'uniswap-v3-deploy-plugin/dist/deployer/UniswapV3Deployer'
import { FeeAmount, TICK_SPACINGS } from '@uniswap/v3-sdk'
import { abi as NFTABI } from '@uniswap/v3-periphery/artifacts/contracts/NonfungiblePositionManager.sol/NonfungiblePositionManager.json'
import { abi as RouterABI } from '@uniswap/v3-periphery/artifacts/contracts/SwapRouter.sol/SwapRouter.json'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'

// Below methods taken from the Uniswap test suite, see
// https://github.com/Uniswap/v3-periphery/blob/main/test/shared/ticks.ts
const getMinTick = (tickSpacing: number) =>
  Math.ceil(-887272 / tickSpacing) * tickSpacing
const getMaxTick = (tickSpacing: number) =>
  Math.floor(887272 / tickSpacing) * tickSpacing

describe('Contract interactions', () => {
  let env: OptimismEnv

  let Factory__ERC20: ContractFactory

  let otherWallet: Wallet

  before(async () => {
    env = await OptimismEnv.new()

    Factory__ERC20 = await ethers.getContractFactory('ERC20', env.l2Wallet)

    otherWallet = Wallet.createRandom().connect(env.l2Wallet.provider)
    await env.l2Wallet.sendTransaction({
      to: otherWallet.address,
      value: utils.parseEther('0.1'),
    })
  })

  describe('ERC20s', () => {
    let contract: Contract

    before(async () => {
      Factory__ERC20 = await ethers.getContractFactory('ERC20', env.l2Wallet)
    })

    it('should successfully deploy the contract', async () => {
      contract = await Factory__ERC20.deploy(100000000, 'OVM Test', 8, 'OVM')
      await contract.deployed()
    })

    it('should support approvals', async () => {
      const spender = '0x' + '22'.repeat(20)
      const tx = await contract.approve(spender, 1000)
      await tx.wait()
      let allowance = await contract.allowance(env.l2Wallet.address, spender)
      expect(allowance).to.deep.equal(BigNumber.from(1000))
      allowance = await contract.allowance(otherWallet.address, spender)
      expect(allowance).to.deep.equal(BigNumber.from(0))

      const logs = await contract.queryFilter(
        contract.filters.Approval(env.l2Wallet.address),
        1,
        'latest'
      )
      expect(logs[0].args._owner).to.equal(env.l2Wallet.address)
      expect(logs[0].args._spender).to.equal(spender)
      expect(logs[0].args._value).to.deep.equal(BigNumber.from(1000))
    })

    it('should support transferring balances', async () => {
      const tx = await contract.transfer(otherWallet.address, 1000)
      await tx.wait()
      const balFrom = await contract.balanceOf(env.l2Wallet.address)
      const balTo = await contract.balanceOf(otherWallet.address)
      expect(balFrom).to.deep.equal(BigNumber.from(100000000).sub(1000))
      expect(balTo).to.deep.equal(BigNumber.from(1000))

      const logs = await contract.queryFilter(
        contract.filters.Transfer(env.l2Wallet.address),
        1,
        'latest'
      )
      expect(logs[0].args._from).to.equal(env.l2Wallet.address)
      expect(logs[0].args._to).to.equal(otherWallet.address)
      expect(logs[0].args._value).to.deep.equal(BigNumber.from(1000))
    })

    it('should support being self destructed', async () => {
      const tx = await contract.destroy()
      await tx.wait()
      const code = await env.l2Wallet.provider.getCode(contract.address)
      expect(code).to.equal('0x')
    })
  })

  describe('uniswap', () => {
    let contracts: { [name: string]: Contract }
    let tokens: Contract[]

    before(async () => {
      if (
        process.env.UNISWAP_POSITION_MANAGER_ADDRESS &&
        process.env.UNISWAP_ROUTER_ADDRESS
      ) {
        console.log('Using predeployed Uniswap. Addresses:')
        console.log(
          `Position manager: ${process.env.UNISWAP_POSITION_MANAGER_ADDRESS}`
        )
        console.log(`Router:           ${process.env.UNISWAP_ROUTER_ADDRESS}`)
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
      }

      const tokenA = await Factory__ERC20.deploy(100000000, 'OVM1', 8, 'OVM1')
      await tokenA.deployed()
      const tokenB = await Factory__ERC20.deploy(100000000, 'OVM2', 8, 'OVM2')
      await tokenB.deployed()
      tokens = [tokenA, tokenB]
      tokens.sort((a, b) => {
        if (a.address > b.address) {
          return 1
        }

        if (a.address < b.address) {
          return -1
        }

        return 0
      })

      const tx = await tokens[0].transfer(otherWallet.address, 100000)
      await tx.wait()
    })

    it('should deploy the Uniswap ecosystem', async function () {
      if (contracts) {
        console.log(
          'Skipping Uniswap deployment since addresses are already defined.'
        )
        this.skip()
        return
      }

      contracts = await UniswapV3Deployer.deploy(env.l2Wallet)
    })

    it('should deploy and initialize a liquidity pool', async () => {
      const tx =
        await contracts.positionManager.createAndInitializePoolIfNecessary(
          tokens[0].address,
          tokens[1].address,
          FeeAmount.MEDIUM,
          // initial ratio of 1/1
          BigNumber.from('79228162514264337593543950336')
        )
      await tx.wait()
    })

    it('should approve the contracts', async () => {
      for (const wallet of [env.l2Wallet, otherWallet]) {
        for (const token of tokens) {
          let tx = await token
            .connect(wallet)
            .approve(contracts.positionManager.address, 100000000)
          await tx.wait()
          tx = await token
            .connect(wallet)
            .approve(contracts.router.address, 100000000)
          await tx.wait()
        }
      }
    })

    it('should mint new positions', async () => {
      const tx = await contracts.positionManager.mint(
        {
          token0: tokens[0].address,
          token1: tokens[1].address,
          tickLower: getMinTick(TICK_SPACINGS[FeeAmount.MEDIUM]),
          tickUpper: getMaxTick(TICK_SPACINGS[FeeAmount.MEDIUM]),
          fee: FeeAmount.MEDIUM,
          recipient: env.l2Wallet.address,
          amount0Desired: 15,
          amount1Desired: 15,
          amount0Min: 0,
          amount1Min: 0,
          deadline: Date.now() * 2,
        },
        {
          gasLimit: 10000000,
        }
      )
      await tx.wait()
      expect(
        await contracts.positionManager.balanceOf(env.l2Wallet.address)
      ).to.eq(1)
      expect(
        await contracts.positionManager.tokenOfOwnerByIndex(
          env.l2Wallet.address,
          0
        )
      ).to.eq(1)
    })

    it('should swap', async () => {
      const tx = await contracts.router.connect(otherWallet).exactInputSingle(
        {
          tokenIn: tokens[0].address,
          tokenOut: tokens[1].address,
          fee: FeeAmount.MEDIUM,
          recipient: otherWallet.address,
          deadline: Date.now() * 2,
          amountIn: 10,
          amountOutMinimum: 0,
          sqrtPriceLimitX96: 0,
        },
        {
          gasLimit: 10000000,
        }
      )
      await tx.wait()
      expect(await tokens[1].balanceOf(otherWallet.address)).to.deep.equal(
        BigNumber.from('5')
      )
    })
  })
})
