import { ethers } from 'ethers'
import { abi as UNISWAP_POOL_ABI } from '@uniswap/v3-core/artifacts/contracts/UniswapV3Pool.sol/UniswapV3Pool.json'
import { UNISWAP_V3_NFPM_ADDRESS } from '../scripts/constants'
import { getUniswapV3Factory, replaceWETH } from '../scripts/utils'
import { expect, env, ERC20_ABI } from './setup'
import { AccountType } from '../scripts/types'

describe('uniswap contracts', () => {
  before(async () => {
    await env.init()
  })

  it('V3 factory', () => {
    if (!env.hasLiveProviders()) {
      console.log('Cannot run factory tests without live provider')
      return
    }

    let preUniswapV3Factory: ethers.Contract
    let postUniswapV3Factory: ethers.Contract
    before(async () => {
      preUniswapV3Factory = getUniswapV3Factory(env.preL2Provider)
      postUniswapV3Factory = getUniswapV3Factory(env.postL2Provider)
    })

    it('should have the same owner', async () => {
      if (!env.hasLiveProviders()) {
        console.log('Cannot run factory tests without live provider')
        return
      }

      const preOwner = await preUniswapV3Factory.owner()
      const postOwner = await postUniswapV3Factory.owner()
      expect(preOwner).to.equal(postOwner)
    })

    it('should have the same feeAmountTickSpacing map values', async () => {
      if (!env.hasLiveProviders()) {
        console.log('Cannot run factory tests without live provider')
        return
      }

      for (const fee of [500, 3000, 10000]) {
        const preValue = await preUniswapV3Factory.feeAmountTickSpacing(fee)
        const postValue = await postUniswapV3Factory.feeAmountTickSpacing(fee)
        expect(preValue).to.deep.equal(postValue)
      }
    })

    it('should have the right pool addresses', async () => {
      if (!env.hasLiveProviders()) {
        console.log('Cannot run factory tests without live provider')
        return
      }

      for (const pool of env.surgeryDataSources.pools) {
        const remotePoolAddress1 = await postUniswapV3Factory.getPool(
          pool.token0,
          pool.token1,
          pool.fee
        )
        const remotePoolAddress2 = await postUniswapV3Factory.getPool(
          pool.token1,
          pool.token0,
          pool.fee
        )
        expect(remotePoolAddress1).to.equal(remotePoolAddress2)
        expect(remotePoolAddress1.toLowerCase()).to.equal(
          pool.newAddress.toLowerCase()
        )
      }
    })

    // Debug this one...
    it('should have the same code as on mainnet', async () => {
      let l2Code = await env.postL2Provider.getCode(
        postUniswapV3Factory.address
      )
      l2Code = replaceWETH(l2Code)
      const l1Code = await env.surgeryDataSources.ethProvider.getCode(
        postUniswapV3Factory.address
      )
      expect(l2Code).to.not.equal('0x')
      expect(l2Code).to.equal(l1Code)
    })
  })

  describe('V3 NFPM', () => {
    it('should have the same code as on mainnet', async () => {
      const l2Code = await env.postL2Provider.getCode(UNISWAP_V3_NFPM_ADDRESS)
      let l1Code = await env.surgeryDataSources.ethProvider.getCode(
        UNISWAP_V3_NFPM_ADDRESS
      )
      l1Code = replaceWETH(l1Code)
      expect(l2Code).to.not.equal('0x')
      expect(l2Code).to.equal(l1Code)
    })

    // TODO: what's the best way to test the _poolIds change?
  })

  describe('V3 pools', () => {
    it('Pools code', () => {
      for (const pool of env.surgeryDataSources.pools) {
        describe(`pool at address ${pool.newAddress}`, () => {
          it('should have the same code as on testnet', async () => {
            const l2Code = await env.postL2Provider.getCode(pool.newAddress)
            const l1Code = await env.surgeryDataSources.ropstenProvider.getCode(
              pool.newAddress
            )
            expect(l2Code).to.not.equal('0x')
            expect(l2Code).to.equal(l1Code)
          })
        })
      }
    })

    it('Pools contract', () => {
      if (!env.hasLiveProviders()) {
        console.log('Cannot run pool contract tests without live provider')
        return
      }
      for (const pool of env.surgeryDataSources.pools) {
        describe(`pool at address ${pool.newAddress}`, () => {
          let prePoolContract: ethers.Contract
          let postPoolContract: ethers.Contract
          before(async () => {
            prePoolContract = new ethers.Contract(
              pool.oldAddress,
              UNISWAP_POOL_ABI,
              env.preL2Provider
            )
            postPoolContract = new ethers.Contract(
              pool.newAddress,
              UNISWAP_POOL_ABI,
              env.postL2Provider
            )
          })

          it('should have the same code as on testnet', async () => {
            const l2Code = await env.postL2Provider.getCode(
              postPoolContract.address
            )
            const l1Code = await env.surgeryDataSources.ethProvider.getCode(
              postPoolContract.address
            )
            expect(l2Code).to.not.equal('0x')
            expect(l2Code).to.equal(l1Code)
          })

          it('should have the same storage values', async () => {
            const varsToCheck = [
              'slot0',
              'feeGrowthGlobal0X128',
              'feeGrowthGlobal1X128',
              'protocolFees',
              'liquidity',
              'factory',
              'token0',
              'token1',
              'fee',
              'tickSpacing',
              'maxLiquidityPerTick',
            ]

            for (const varName of varsToCheck) {
              const preValue = await prePoolContract[varName]({
                blockTag: env.config.stateDumpHeight,
              })
              const postValue = await postPoolContract[varName]()
              expect(preValue).to.deep.equal(postValue)
            }
          })

          it('should have the same token balances as before', async () => {
            const baseERC20 = new ethers.Contract(
              ethers.constants.AddressZero,
              ERC20_ABI
            )
            const preToken0 = baseERC20
              .attach(pool.token0)
              .connect(env.preL2Provider)
            const postToken0 = baseERC20
              .attach(pool.token0)
              .connect(env.postL2Provider)
            const preToken1 = baseERC20
              .attach(pool.token1)
              .connect(env.preL2Provider)
            const postToken1 = baseERC20
              .attach(pool.token1)
              .connect(env.postL2Provider)

            // Token0 might not have any code in the new system, we can skip this check if so.
            const newToken0Code = await env.postL2Provider.getCode(pool.token0)
            if (newToken0Code !== '0x') {
              const preBalance0 = await preToken0.balanceOf(pool.oldAddress, {
                blockTag: env.config.stateDumpHeight,
              })
              const postBalance0 = await postToken0.balanceOf(pool.newAddress)
              expect(preBalance0).to.deep.equal(postBalance0)
            }

            // Token1 might not have any code in the new system, we can skip this check if so.
            const newToken1Code = await env.postL2Provider.getCode(pool.token1)
            if (newToken1Code !== '0x') {
              const preBalance1 = await preToken1.balanceOf(pool.oldAddress, {
                blockTag: env.config.stateDumpHeight,
              })
              const postBalance1 = await postToken1.balanceOf(pool.newAddress)
              expect(preBalance1).to.deep.equal(postBalance1)
            }
          })
        })
      }

      // TODO: add a test for minting positions?
    })
  })

  describe('other', () => {
    let accs
    before(async () => {
      accs = env.getAccountsByType(AccountType.UNISWAP_V3_OTHER)
    })

    // TODO: for some reason these tests fail
    it('Other uniswap contracts', () => {
      for (const acc of accs) {
        describe(`uniswap contract at address ${acc.address}`, () => {
          it('should have the same code as on mainnet', async () => {
            const l2Code = await env.postL2Provider.getCode(acc.address)
            let l1Code = await env.surgeryDataSources.ethProvider.getCode(
              acc.address
            )
            l1Code = replaceWETH(l1Code)
            expect(l2Code).to.not.equal('0x')
            expect(l2Code).to.equal(l1Code)
          })
        })
      }
    })
  })
})
