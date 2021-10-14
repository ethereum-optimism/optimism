/* eslint-disable @typescript-eslint/no-empty-function */
import { expect, env, NUM_ACCOUNTS_DIVISOR } from '../setup'
import { AccountType, Account } from '../../scripts/types'

describe('verified', () => {
  before(async () => {
    const verified = env.getAccountsByType(AccountType.VERIFIED)

    for (const [i, account] of verified.entries()) {
      if (i % NUM_ACCOUNTS_DIVISOR === 0) {
        const preBytecode = await env.preL2Provider.getCode(account.address)
        const postBytecode = await env.postL2Provider.getCode(account.address)

        describe(`account ${i}/${verified.length} (${account.address})`, () => {
          it('should have new bytecode with equal or smaller size', async () => {
            const preSize = preBytecode.length
            const postSize = postBytecode.length
            expect(preSize >= postSize).to.be.true
          })

          it('should have the same nonce and balance', async () => {
            const preNonce = await env.preL2Provider.getTransactionCount(
              account.address,
              env.config.stateDumpHeight
            )
            const postNonce = await env.postL2Provider.getTransactionCount(
              account.address
            )
            expect(preNonce).to.deep.eq(postNonce)

            const preBalance = await env.preL2Provider.getBalance(
              account.address,
              env.config.stateDumpHeight
            )
            const postBalance = await env.postL2Provider.getBalance(
              account.address
            )
            expect(preBalance).to.deep.eq(postBalance)
          })
        })
      }
    }
  })

  it('stub', async () => {})
})
