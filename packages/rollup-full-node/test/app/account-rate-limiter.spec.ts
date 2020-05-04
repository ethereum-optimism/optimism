/* External Imports */
import { sleep, TestUtils } from '@eth-optimism/core-utils'

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
    accountRateLimiter = new DefaultAccountRateLimiter(
      1,
      1,
      1_000,
      1_000,
      1_000
    )
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

  it('rate limits transactions if outside of range', async () => {
    // Should not throw
    accountRateLimiter.validateTransactionRateLimit(addressOne)
    TestUtils.assertThrows(() => {
      accountRateLimiter.validateTransactionRateLimit(addressOne)
    }, TransactionLimitError)

    // Should not throw
    accountRateLimiter.validateTransactionRateLimit(addressTwo)

    await sleep(1_050)
    // should not throw anymore
    accountRateLimiter.validateTransactionRateLimit(addressOne)
  })

  it('rate limits requests if outside of range', async () => {
    // Should not throw
    accountRateLimiter.validateRateLimit(ipOne)
    TestUtils.assertThrows(() => {
      accountRateLimiter.validateRateLimit(ipOne)
    }, RateLimitError)

    // Should not throw
    accountRateLimiter.validateRateLimit(ipTwo)

    await sleep(1_050)
    // should not throw anymore
    accountRateLimiter.validateRateLimit(ipOne)
  })

  describe('Environment Variable Refresh -- no change', () => {
    it('post-refresh: does not rate limit transactions if in range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      accountRateLimiter.validateTransactionRateLimit(addressTwo)
    })

    it('post-refresh: does not rate limit requests if in range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateRateLimit(ipOne)
      accountRateLimiter.validateRateLimit(ipTwo)
    })

    it('post-refresh: rate limits transactions if outside of range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateTransactionRateLimit(addressOne)
      }, TransactionLimitError)

      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressTwo)

      await sleep(1_050)
      // should not throw anymore
      accountRateLimiter.validateTransactionRateLimit(addressOne)
    })

    it('post-refresh: rate limits requests if outside of range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateRateLimit(ipOne)
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateRateLimit(ipOne)
      }, RateLimitError)

      // Should not throw
      accountRateLimiter.validateRateLimit(ipTwo)

      await sleep(1_050)
      // should not throw anymore
      accountRateLimiter.validateRateLimit(ipOne)
    })
  })

  describe('Environment Variable Refresh -- duration increased', () => {
    beforeEach(() => {
      process.env.REQUEST_LIMIT_PERIOD_MILLIS = '3000'
    })
    afterEach(() => {
      delete process.env.REQUEST_LIMIT_PERIOD_MILLIS
    })

    it('post-refresh: does not rate limit transactions if in range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      accountRateLimiter.validateTransactionRateLimit(addressTwo)
    })

    it('post-refresh: does not rate limit requests if in range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateRateLimit(ipOne)
      accountRateLimiter.validateRateLimit(ipTwo)
    })

    it('post-refresh: rate limits transactions if outside of range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateTransactionRateLimit(addressOne)
      }, TransactionLimitError)

      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressTwo)

      await sleep(1_050)
      // should still throw
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateTransactionRateLimit(addressOne)
      }, TransactionLimitError)
    })

    it('post-refresh: rate limits requests if outside of range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateRateLimit(ipOne)
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateRateLimit(ipOne)
      }, RateLimitError)

      // Should not throw
      accountRateLimiter.validateRateLimit(ipTwo)

      await sleep(1_050)
      // should still throw
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateRateLimit(ipOne)
      }, RateLimitError)
    })
  })

  describe('Environment Variable Refresh -- tx limit increased', () => {
    beforeEach(() => {
      process.env.MAX_TRANSACTIONS_PER_UNIT_TIME = '2'
    })
    afterEach(() => {
      delete process.env.MAX_TRANSACTIONS_PER_UNIT_TIME
    })

    it('post-refresh: does not rate limit transactions if in range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      accountRateLimiter.validateTransactionRateLimit(addressTwo)
      accountRateLimiter.validateTransactionRateLimit(addressTwo)
    })

    it('post-refresh: does not rate limit requests if in range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateRateLimit(ipOne)
      accountRateLimiter.validateRateLimit(ipTwo)
    })

    it('post-refresh: rate limits transactions if outside of range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateTransactionRateLimit(addressOne)
      }, TransactionLimitError)

      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressTwo)
    })

    it('post-refresh: rate limits requests if outside of range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateRateLimit(ipOne)
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateRateLimit(ipOne)
      }, RateLimitError)

      // Should not throw
      accountRateLimiter.validateRateLimit(ipTwo)
    })
  })

  describe('Environment Variable Refresh -- request limit increased', () => {
    beforeEach(() => {
      process.env.MAX_NON_TRANSACTION_REQUESTS_PER_UNIT_TIME = '2'
    })
    afterEach(() => {
      delete process.env.MAX_NON_TRANSACTION_REQUESTS_PER_UNIT_TIME
    })

    it('post-refresh: does not rate limit transactions if in range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      accountRateLimiter.validateTransactionRateLimit(addressTwo)
    })

    it('post-refresh: does not rate limit requests if in range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateRateLimit(ipOne)
      accountRateLimiter.validateRateLimit(ipOne)
      accountRateLimiter.validateRateLimit(ipTwo)
      accountRateLimiter.validateRateLimit(ipTwo)
    })

    it('post-refresh: rate limits transactions if outside of range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressOne)
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateTransactionRateLimit(addressOne)
      }, TransactionLimitError)

      // Should not throw
      accountRateLimiter.validateTransactionRateLimit(addressTwo)
    })

    it('post-refresh: rate limits requests if outside of range', async () => {
      await sleep(2_000)
      // Should not throw
      accountRateLimiter.validateRateLimit(ipOne)
      accountRateLimiter.validateRateLimit(ipOne)
      TestUtils.assertThrows(() => {
        accountRateLimiter.validateRateLimit(ipOne)
      }, RateLimitError)

      // Should not throw
      accountRateLimiter.validateRateLimit(ipTwo)
    })
  })
})
