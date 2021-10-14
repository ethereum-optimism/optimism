import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'
import { add0x } from '@eth-optimism/core-utils'
import { expect, env, NUM_ACCOUNTS_DIVISOR } from '../setup'
import { AccountType, Account } from '../../scripts/types'

describe('EOAs', () => {
  describe('standard EOA', () => {
    before(async () => {
      const eoas = env.getAccountsByType(AccountType.EOA)

      for (const [i, eoa] of eoas.entries()) {
        if (i % NUM_ACCOUNTS_DIVISOR === 0) {
          describe(`account ${i}/${eoas.length} (${eoa.address})`, () => {
            it('should not have any code', async () => {
              const code = await env.postL2Provider.getCode(eoa.address)
              expect(code).to.eq('0x')
            })

            it('should have the null code hash and storage root', async () => {
              const proof = await env.postL2Provider.send('eth_getProof', [
                eoa.address,
                [],
                'latest',
              ])

              expect(proof.codeHash).to.equal(add0x(KECCAK256_NULL_S))
              expect(proof.storageHash).to.equal(add0x(KECCAK256_RLP_S))
            })

            it('should have the same balance as it had before', async () => {
              // Balance before needs to come from the specific block at which the dump was taken.
              const preBalance = await env.preL2Provider.getBalance(
                eoa.address,
                env.config.stateDumpHeight
              )

              // Balance after can come from the latest block.
              const postBalance = await env.postL2Provider.getBalance(
                eoa.address
              )

              expect(preBalance).to.deep.eq(postBalance)
            })

            it('should have the same nonce as it had before', async () => {
              // Nonce before needs to come from the specific block at which the dump was taken.
              const preNonce = await env.preL2Provider.getTransactionCount(
                eoa.address,
                env.config.stateDumpHeight
              )

              // Nonce after can come from the latest block.
              const postNonce = await env.postL2Provider.getTransactionCount(
                eoa.address
              )

              expect(preNonce).to.deep.eq(postNonce)
            })
          })
        }
      }
    })

    // Hack for dynamically generating tests based on async data.
    // eslint-disable-next-line @typescript-eslint/no-empty-function
    it('stub', async () => {})
  })

  // Does not exist on Kovan?
  describe.skip('1inch deployer', () => {
    let eoa: Account
    before(() => {
      eoa = env.getAccountsByType(AccountType.ONEINCH_DEPLOYER)[0]
    })

    it('should not have any code', async () => {
      const code = await env.postL2Provider.getCode(eoa.address)
      expect(code).to.eq('0x')
    })

    it('should have the null code hash and storage root', async () => {
      const proof = await env.postL2Provider.send('eth_getProof', [
        eoa.address,
        [],
        'latest',
      ])

      expect(proof.codeHash).to.equal(add0x(KECCAK256_NULL_S))
      expect(proof.storageHash).to.equal(add0x(KECCAK256_RLP_S))
    })

    it('should have the same balance as it had before', async () => {
      // Balance before needs to come from the specific block at which the dump was taken.
      const preBalance = await env.preL2Provider.getBalance(
        eoa.address,
        env.config.stateDumpHeight
      )

      // Balance after can come from the latest block.
      const postBalance = await env.postL2Provider.getBalance(eoa.address)

      expect(preBalance).to.deep.eq(postBalance)
    })

    it('should have a nonce equal to zero', async () => {
      // Nonce before needs to come from the specific block at which the dump was taken.
      const preNonce = await env.preL2Provider.getTransactionCount(
        eoa.address,
        env.config.stateDumpHeight
      )

      // Nonce after can come from the latest block.
      const postNonce = await env.postL2Provider.getTransactionCount(
        eoa.address
      )

      expect(preNonce).to.deep.eq(postNonce)
    })
  })
})
