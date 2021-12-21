import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'
import { add0x } from '@eth-optimism/core-utils'
import { expect, env } from './setup'
import { AccountType, Account } from '../scripts/types'

describe('EOAs', () => {
  describe('standard EOA', () => {
    let eoas
    before(async () => {
      await env.init()
      eoas = env.getAccountsByType(AccountType.EOA)
    })

    it('EOAs', () => {
      for (const [i, eoa] of eoas.entries()) {
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
            const postBalance = await env.postL2Provider.getBalance(eoa.address)

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
    })
  })

  // eslint-disable-next-line
  describe('1inch deployer', function() {
    let eoa: Account
    // eslint-disable-next-line
    before(function() {
      if (env.surgeryDataSources.configs.l2NetworkName === 'kovan') {
        console.log('1inch deployer does not exist on Optimism Kovan')
        this.skip()
      }

      eoa = env.getAccountsByType(AccountType.ONEINCH_DEPLOYER)[0]
      if (!eoa) {
        throw new Error('Cannot find one inch deployer')
      }
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
      expect(preNonce).to.not.eq(0)

      // Nonce after can come from the latest block.
      const postNonce = await env.postL2Provider.getTransactionCount(
        eoa.address
      )
      expect(postNonce).to.deep.eq(0)
    })
  })
})
