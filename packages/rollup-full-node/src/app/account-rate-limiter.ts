/* External imports */
import {
  getLogger,
  logError,
  TimeBucketedCounter,
} from '@eth-optimism/core-utils'

/* Internal imports */
import {
  AccountRateLimiter,
  RateLimitError,
  TransactionLimitError,
} from '../types'
import { Environment } from './util'

const log = getLogger('routing-handler')

export class NoOpAccountRateLimiter implements AccountRateLimiter {
  public validateRateLimit(sourceIpAddress: string): void {
    /* no-op */
  }
  public validateTransactionRateLimit(address: string): void {
    /* no-op */
  }
}

/**
 * Keeps track of and enforces rate limits for accounts by address and/or IP address.
 */
export class DefaultAccountRateLimiter implements AccountRateLimiter {
  private readonly ipToRequestCounter: Map<string, TimeBucketedCounter>
  private readonly addressToRequestCounter: Map<string, TimeBucketedCounter>

  private readonly requestingIpsSinceLastPurge: Set<string>
  private readonly requestingAddressesSinceLastPurge: Set<string>

  constructor(
    private maxRequestsPerTimeUnit,
    private maxTransactionsPerTimeUnit,
    private requestLimitPeriodInMillis,
    purgeIntervalMultiplier: number = 1_000,
    variableRefreshRateMillis = 300_000
  ) {
    this.requestingIpsSinceLastPurge = new Set<string>()
    this.requestingAddressesSinceLastPurge = new Set<string>()

    this.ipToRequestCounter = new Map<string, TimeBucketedCounter>()
    this.addressToRequestCounter = new Map<string, TimeBucketedCounter>()

    setInterval(() => {
      this.purgeStale(false)
    }, this.requestLimitPeriodInMillis * (purgeIntervalMultiplier - 5))

    setInterval(() => {
      this.purgeStale(true)
    }, this.requestLimitPeriodInMillis * (purgeIntervalMultiplier + 5))

    setInterval(() => {
      this.refreshVariables()
    }, variableRefreshRateMillis)
  }

  /**
   * Validates the rate limit for the provided IP address, incrementing the total to account for this request.
   *
   * @param sourceIpAddress The requesting IP address.
   * @throws RateLimitError if this request is above the rate limit threshold
   */
  public validateRateLimit(sourceIpAddress: string): void {
    if (!this.ipToRequestCounter.has(sourceIpAddress)) {
      this.ipToRequestCounter.set(
        sourceIpAddress,
        new TimeBucketedCounter(this.requestLimitPeriodInMillis, 10)
      )
    }

    this.requestingIpsSinceLastPurge.add(sourceIpAddress)

    const numRequests = this.ipToRequestCounter.get(sourceIpAddress).increment()
    if (
      this.maxRequestsPerTimeUnit !== undefined &&
      numRequests > this.maxRequestsPerTimeUnit
    ) {
      throw new RateLimitError(
        sourceIpAddress,
        numRequests,
        this.maxRequestsPerTimeUnit,
        this.requestLimitPeriodInMillis
      )
    }
  }

  /**
   * Validates the rate limit for the provided Ethereum Address.
   *
   * @param address The Ethereum address of the request.
   * @throws TransactionLimitError if this request puts the account above the rate limit threshold.
   */
  public validateTransactionRateLimit(address: string): void {
    if (!this.addressToRequestCounter.has(address)) {
      this.addressToRequestCounter.set(
        address,
        new TimeBucketedCounter(this.requestLimitPeriodInMillis, 10)
      )
    }

    this.requestingAddressesSinceLastPurge.add(address)

    const numRequests = this.addressToRequestCounter.get(address).increment()
    if (
      this.maxTransactionsPerTimeUnit !== undefined &&
      numRequests > this.maxTransactionsPerTimeUnit
    ) {
      throw new TransactionLimitError(
        address,
        numRequests,
        this.maxTransactionsPerTimeUnit,
        this.requestLimitPeriodInMillis
      )
    }
  }

  /**
   * Purges stale entries from the counters so memory doesn't continue to expand.
   *
   * @param addresses Whether or not to purge the address map.
   */
  private purgeStale(addresses: boolean = true): void {
    let map: Map<string, TimeBucketedCounter>
    let set: Set<string>
    if (addresses) {
      map = this.addressToRequestCounter
      set = this.requestingAddressesSinceLastPurge
    } else {
      map = this.ipToRequestCounter
      set = this.requestingIpsSinceLastPurge
    }

    map.forEach((value, key) => {
      if (!set.has(key)) {
        map.delete(key)
      }
    })
    set.clear()
  }

  /**
   * Refreshes configured member variables from updated Environment Variables.
   */
  private refreshVariables(): void {
    try {
      const envPeriod = Environment.requestLimitPeriodMillis()
      if (this.requestLimitPeriodInMillis !== envPeriod) {
        const prevVal = this.requestLimitPeriodInMillis
        this.requestLimitPeriodInMillis = envPeriod
        this.ipToRequestCounter.clear()
        this.addressToRequestCounter.clear()
        this.requestingIpsSinceLastPurge.clear()
        this.requestingAddressesSinceLastPurge.clear()
        log.info(
          `Updated Rate Limit time period from ${prevVal} to ${this.requestLimitPeriodInMillis} millis`
        )
      }

      const envRequestLimit = Environment.maxNonTransactionRequestsPerUnitTime()
      if (this.maxRequestsPerTimeUnit !== envRequestLimit) {
        const prevVal = this.maxRequestsPerTimeUnit
        this.maxRequestsPerTimeUnit = envRequestLimit
        log.info(
          `Updated Max Requests Per unit time value from ${prevVal} to ${this.maxRequestsPerTimeUnit}`
        )
      }

      const envTxLimit = Environment.maxTransactionsPerUnitTime()
      if (!!envTxLimit && this.maxTransactionsPerTimeUnit !== envTxLimit) {
        const prevVal = this.maxTransactionsPerTimeUnit
        this.maxTransactionsPerTimeUnit = envTxLimit
        log.info(
          `Updated Max Transactions Per unit time value from ${prevVal} to ${this.maxTransactionsPerTimeUnit}`
        )
      }
    } catch (e) {
      logError(
        log,
        `Error updating rate limiter variables from environment variables`,
        e
      )
    }
  }
}
