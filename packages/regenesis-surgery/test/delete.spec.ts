import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'
import { add0x } from '@eth-optimism/core-utils'
import { ethers } from 'ethers'

import { expect, env } from './setup'
import { AccountType } from '../scripts/types'

describe('deleted contracts', () => {
  let accs
  before(async () => {
    await env.init()
    accs = env.getAccountsByType(AccountType.DELETE)
  })

  it('accounts', async () => {
    for (const [i, acc] of accs.entries()) {
      describe(`account ${i}/${accs.length} (${acc.address})`, () => {
        it('should not have any code', async () => {
          const code = await env.postL2Provider.getCode(acc.address)
          expect(code).to.eq('0x')
        })

        it('should have the null code hash and storage root', async () => {
          const proof = await env.postL2Provider.send('eth_getProof', [
            acc.address,
            [],
            'latest',
          ])

          expect(proof.codeHash).to.equal(add0x(KECCAK256_NULL_S))

          expect(proof.storageHash).to.equal(add0x(KECCAK256_RLP_S))
        })

        it('should have a balance equal to zero', async () => {
          // Balance after can come from the latest block.
          const balance = await env.postL2Provider.getBalance(acc.address)

          expect(balance).to.deep.eq(ethers.BigNumber.from(0))
        })

        it('should have a nonce equal to zero', async () => {
          // Nonce after can come from the latest block.
          const nonce = await env.postL2Provider.getTransactionCount(
            acc.address
          )

          expect(nonce).to.deep.eq(0)
        })
      })
    }
  })
})
