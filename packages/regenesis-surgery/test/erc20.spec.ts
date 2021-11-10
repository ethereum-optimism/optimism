import { expect } from '@eth-optimism/core-utils/test/setup'
import { BigNumber } from 'ethers'
import { env } from './setup'

describe('erc20', () => {
  describe('standard ERC20', () => {
    before(async () => {
      await env.init()
    })

    it('ERC20s', () => {
      for (const [i, erc20] of env.erc20s.entries()) {
        describe(`erc20 ${i}/${env.erc20s.length} (${erc20.address})`, () => {
          it('should have the same storage', async () => {
            const account = env.surgeryDataSources.dump.find(
              (a) => a.address === erc20.address
            )
            if (account.storage) {
              for (const key of Object.keys(account.storage)) {
                const pre = await env.preL2Provider.getStorageAt(
                  account.address,
                  BigNumber.from(key)
                )
                const post = await env.postL2Provider.getStorageAt(
                  account.address,
                  BigNumber.from(key)
                )
                expect(pre).to.deep.eq(post)
              }
            }
          })
        })
      }
    })
  })
})
