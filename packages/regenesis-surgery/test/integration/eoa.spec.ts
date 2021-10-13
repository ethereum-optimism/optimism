import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'
import { add0x } from '@eth-optimism/core-utils'
import { expect, env } from '../setup'
import { AccountType, Account } from '../../scripts/types'

// Test 1/X accounts to speed things up.
const NUM_ACCOUNTS_DIVISOR = 1024

describe('EOAs', () => {
  before(async () => {
    await env.init()
  })

  describe('standard EOA', () => {
    let eoas: Account[]
    before(async () => {
      eoas = env.getAccountsByType(AccountType.EOA)
    })

    // Iterate through all of the EOAs and check that they have no code
    // in the new node
    it('should not have any code', async () => {
      for (const [i, eoa] of eoas.entries()) {
        if (i % NUM_ACCOUNTS_DIVISOR !== 0) {
          continue
        }

        console.log(`checking account ${i}/${eoas.length}`)

        const code = await env.postL2Provider.getCode(eoa.address)
        expect(code).to.eq('0x')
      }
    })

    it('should have the null code hash and storage root', async () => {
      for (const [i, eoa] of eoas.entries()) {
        if (i % NUM_ACCOUNTS_DIVISOR !== 0) {
          continue
        }

        console.log(`checking account ${i}/${eoas.length}`)

        const proof = await env.postL2Provider.send('eth_getProof', [
          eoa.address,
          [],
          'latest',
        ])

        expect(proof.codeHash).to.equal(
          add0x(KECCAK256_NULL_S),
          `incorrect code hash for account ${eoa.address}`
        )

        expect(proof.storageHash).to.equal(
          add0x(KECCAK256_RLP_S),
          `incorrect storage root for account ${eoa.address}`
        )
      }
    })

    it('should have the same balance as it had before', async () => {
      for (const [i, eoa] of eoas.entries()) {
        if (i % NUM_ACCOUNTS_DIVISOR !== 0) {
          continue
        }

        console.log(`checking account ${i}/${eoas.length}`)

        // Balance before needs to come from the specific block at which the dump was taken.
        const preBalance = await env.preL2Provider.getBalance(
          eoa.address,
          env.config.stateDumpHeight
        )

        // Balance after can come from the latest block.
        const postBalance = await env.postL2Provider.getBalance(eoa.address)

        expect(preBalance).to.deep.eq(
          postBalance,
          `balance mismatch for address ${eoa.address}`
        )
      }
    })

    it('should have the same nonce as it had before', async () => {
      for (const [i, eoa] of eoas.entries()) {
        if (i % NUM_ACCOUNTS_DIVISOR !== 0) {
          continue
        }

        console.log(`checking account ${i}/${eoas.length}`)

        // Nonce before needs to come from the specific block at which the dump was taken.
        const preNonce = await env.preL2Provider.getTransactionCount(
          eoa.address,
          env.config.stateDumpHeight
        )

        // Nonce after can come from the latest block.
        const postNonce = await env.postL2Provider.getTransactionCount(
          eoa.address
        )

        expect(preNonce).to.deep.eq(
          postNonce,
          `nonce mismatch for address ${eoa.address}`
        )
      }
    })
  })

  // Does not exist on Kovan?
  describe.skip('1inch deployer', () => {
    let eoa: Account
    before(async () => {
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

      expect(proof.codeHash).to.equal(
        add0x(KECCAK256_NULL_S),
        `incorrect code hash for account ${eoa.address}`
      )

      expect(proof.storageHash).to.equal(
        add0x(KECCAK256_RLP_S),
        `incorrect storage root for account ${eoa.address}`
      )
    })

    it('should have the same balance as it had before', async () => {
      // Balance before needs to come from the specific block at which the dump was taken.
      const preBalance = await env.preL2Provider.getBalance(
        eoa.address,
        env.config.stateDumpHeight
      )

      // Balance after can come from the latest block.
      const postBalance = await env.postL2Provider.getBalance(eoa.address)

      expect(preBalance).to.deep.eq(
        postBalance,
        `balance mismatch for address ${eoa.address}`
      )
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

      expect(preNonce).to.deep.eq(
        postNonce,
        `nonce mismatch for address ${eoa.address}`
      )
    })
  })
})
