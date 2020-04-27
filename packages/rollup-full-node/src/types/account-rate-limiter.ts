import { RateLimitError } from './errors'

/**
 * Handles time-based rate limiting by source IP address and/or Ethereum address.
 */
export interface AccountRateLimiter {
  /**
   * Validates the rate limit for the provided IP address, incrementing the total to account for this request.
   *
   * @param sourceIpAddress The requesting IP address.
   * @throws RateLimitError if this request is above the rate limit threshold
   */
  validateRateLimit(sourceIpAddress: string): void

  /**
   * Validates the rate limit for the provided Ethereum Address.
   *
   * @param address The Ethereum address of the request.
   * @throws TransactionLimitError if this request puts the account above the rate limit threshold.
   */
  validateTransactionRateLimit(address: string): void
}
