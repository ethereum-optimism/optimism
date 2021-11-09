/* eslint-disable @typescript-eslint/no-empty-function */
import { expect, env } from './setup'
import { AccountType, Account } from '../scripts/types'

describe('predeploys', () => {
  let predeploys = {
    eth: [],
    newNotEth: [],
    noWipe: [],
    wipe: [],
    weth: [],
  }

  before(async () => {
    await env.init()
    predeploys.eth = env.getAccountsByType(AccountType.PREDEPLOY_ETH)
    predeploys.newNotEth = env.getAccountsByType(AccountType.PREDEPLOY_NEW_NOT_ETH)
    predeploys.noWipe = env.getAccountsByType(AccountType.PREDEPLOY_NO_WIPE)
    predeploys.wipe = env.getAccountsByType(AccountType.PREDEPLOY_WIPE)
    predeploys.weth = env.getAccountsByType(AccountType.PREDEPLOY_WETH)
  })

  it('predeploy tests', () => {
    describe.skip('new predeploys that are not ETH', () => {
      it('should have the exact state specified in the base genesis file', async () => {})
    })

    describe.skip('predeploys where the old state should be wiped', () => {
      it('should have the code and storage of the base genesis file', async () => {})

      it('should have the same nonce and balance as before', async () => {})
    })

    describe.skip('predeploys where the old state should be preserved', () => {
      it('should have the code of the base genesis file', async () => {})

      it('should have the combined storage of the old and new state', async () => {})

      it('should have the same nonce and balance as before', async () => {})
    })

    describe.skip('OVM_ETH', () => {
      it('should have disabled ERC20 features', async () => {})

      it('should no recorded balance for the contracts that move to WETH9', async () => {})

      it('should have a new balance for WETH9 equal to the sum of the moved contract balances', async () => {})
    })

    describe.skip('WETH9', () => {
      it('should have balances for each contract that should move', async () => {})

      it('should have a balance equal to the sum of all moved balances', async () => {})
    })
  })
})
