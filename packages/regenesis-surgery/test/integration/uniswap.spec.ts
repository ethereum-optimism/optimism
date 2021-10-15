import { ethers } from 'ethers'
import { abi as UNISWAP_POOL_ABI } from '@uniswap/v3-core/artifacts/contracts/UniswapV3Pool.sol/UniswapV3Pool.json'
import { UNISWAP_V3_NFPM_ADDRESS } from '../../scripts/constants'
import { getUniswapV3Factory, replaceWETH } from '../../scripts/utils'
import { expect, env } from '../setup'
import { AccountType } from '../../scripts/types'

const ERC20_ABI = ['function balanceOf(address owner) view returns (uint256)']

describe('uniswap contracts', () => {
  describe('V3 factory', () => {
    let preUniswapV3Factory: ethers.Contract
    let postUniswapV3Factory: ethers.Contract
    before(async () => {
      preUniswapV3Factory = getUniswapV3Factory(env.preL2Provider)
      postUniswapV3Factory = getUniswapV3Factory(env.postL2Provider)
    })

    it('should have the same owner', async () => {
      const preOwner = await preUniswapV3Factory.owner()
      const postOwner = await postUniswapV3Factory.owner()
      expect(preOwner).to.equal(postOwner)
    })

    it('should have the same feeAmountTickSpacing map values', async () => {
      for (const fee of [500, 3000, 10000]) {
        const preValue = await preUniswapV3Factory.feeAmountTickSpacing(fee)
        const postValue = await postUniswapV3Factory.feeAmountTickSpacing(fee)
        expect(preValue).to.deep.equal(postValue)
      }
    })

    it('should have the right pool addresses', async () => {
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

    it('should have the same code as on mainnet', async () => {
      const l2Code = await env.postL2Provider.getCode(
        postUniswapV3Factory.address
      )
      const l1Code = await env.surgeryDataSources.l1Provider.getCode(
        postUniswapV3Factory.address
      )
      expect(l2Code).to.not.equal('0x')
      expect(l2Code).to.equal(l1Code)
    })
  })

  describe('V3 NFPM', () => {
    it('should have the same code as on mainnet', async () => {
      let l2Code = await env.postL2Provider.getCode(UNISWAP_V3_NFPM_ADDRESS)
      const l1Code = await env.surgeryDataSources.l1Provider.getCode(
        UNISWAP_V3_NFPM_ADDRESS
      )
      expect(l2Code).to.not.equal('0x')
      l2Code = replaceWETH(l2Code)
      expect(l2Code).to.equal(l1Code)
    })

    // TODO: what's the best way to test the _poolIds change?
  })

  describe('V3 pools', () => {
    before(async () => {
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
            const l1Code = await env.surgeryDataSources.l1Provider.getCode(
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

    // Hack for dynamically generating tests based on async data.
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    it('stub', async () => {})
  })

  describe('other', () => {
    before(async () => {
      const accs = env.getAccountsByType(AccountType.UNISWAP_V3_OTHER)

      for (const acc of accs) {
        describe(`uniswap contract at address ${acc.address}`, () => {
          it('should have the same code as on mainnet', async () => {
            const l2Code = await env.postL2Provider.getCode(acc.address)
            const l1Code = await env.surgeryDataSources.l1Provider.getCode(
              acc.address
            )
            expect(l2Code).to.not.equal('0x')
            expect(l2Code).to.equal(l1Code)
          })
        })
      }
    })

    // Hack for dynamically generating tests based on async data.
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    it('stub', async () => {})
  })
})
