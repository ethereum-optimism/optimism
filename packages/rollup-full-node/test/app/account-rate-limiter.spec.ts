/* External Imports */
import { TestUtils } from '@eth-optimism/core-utils'

import { Wallet } from 'ethers'

/* Internal Imports */
import {
  AccountRateLimiter,
  RateLimitError,
  TransactionLimitError,
} from '../../src/types'
import { DefaultAccountRateLimiter } from '../../src/app'

describe('Account Rate Limiter', () => {
  const addressOne: string = Wallet.createRandom().address
  const addressTwo: string = Wallet.createRandom().address
  const ipOne: string = '0.0.0.0'
  const ipTwo: string = '0.0.0.1'
  let accountRateLimiter: AccountRateLimiter

  beforeEach(() => {
    accountRateLimiter = new DefaultAccountRateLimiter(1, 1, 10_000)
  })

  it('does not rate limit transactions if in range', () => {
    // Should not throw
    accountRateLimiter.validateTransactionRateLimit(addressOne)
    accountRateLimiter.validateTransactionRateLimit(addressTwo)
  })

  it('does not rate limit requests if in range', () => {
    // Should not throw
    accountRateLimiter.validateRateLimit(ipOne)
    accountRateLimiter.validateRateLimit(ipTwo)
  })

  it('rate limits transactions if outside of range', () => {
    // Should not throw
    accountRateLimiter.validateTransactionRateLimit(addressOne)
    TestUtils.assertThrows(() => {
      accountRateLimiter.validateTransactionRateLimit(addressOne)
    }, TransactionLimitError)

    // Should not throw
    accountRateLimiter.validateTransactionRateLimit(addressTwo)
  })

  it('rate limits requests if outside of range', () => {
    // Should not throw
    accountRateLimiter.validateRateLimit(ipOne)
    TestUtils.assertThrows(() => {
      accountRateLimiter.validateRateLimit(ipOne)
    }, RateLimitError)

    // Should not throw
    accountRateLimiter.validateRateLimit(ipTwo)
  })
})
