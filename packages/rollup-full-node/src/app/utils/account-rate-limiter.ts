/* External imports */
import { getLogger, TimeBucketedCounter } from '@eth-optimism/core-utils'

/* Internal imports */
import { RateLimitError, TransactionLimitError } from '../../types'
const log = getLogger('routing-handler')

/**
 * Keeps track of and enforces rate limits for accounts by address and/or IP address.
 */
export class AccountRateLimiter {
  private readonly ipToRequestCounter: Map<string, TimeBucketedCounter>
  private readonly addressToRequestCounter: Map<string, TimeBucketedCounter>

  private readonly requestingIpsSinceLastPurge: Set<string>
  private readonly requestingAddressesSinceLastPurge: Set<string>

  constructor(
    private readonly maxRequestsPerTimeUnit,
    private readonly maxTransactionsPerTimeUnit,
    private readonly requestLimitPeriodInMillis,
    private purgeIntervalMultiplier: number = 1_000
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
    if (numRequests > this.maxRequestsPerTimeUnit) {
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
    if (numRequests > this.maxRequestsPerTimeUnit) {
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
    let map
    let set
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
  }
}
