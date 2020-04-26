/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import { getLogger, logError, SimpleClient } from '@eth-optimism/core-utils'

import { parseTransaction, Transaction } from 'ethers/utils'
/* Internal Imports */
import {
  FullnodeHandler,
  InvalidParametersError,
  InvalidTransactionDesinationError,
  UnsupportedMethodError,
  Web3RpcMethods,
  web3RpcMethodsExcludingTest,
} from '../../types'
import { AccountRateLimiter } from '../utils'

const log = getLogger('routing-handler')

const methodsToRouteWithTransactionHandler: string[] = [
  Web3RpcMethods.sendTransaction,
  Web3RpcMethods.sendRawTransaction,
  Web3RpcMethods.getTransactionByHash,
  Web3RpcMethods.getBlockByNumber,
  Web3RpcMethods.getBlockByHash,
]

/**
 * Handles rate-limiting requests by Ethereum address for transactions and by IP address for all
 * other request types and then routes them according to their method.
 *
 * If they are read-only they'll go to the provided read-only provider, and
 * otherwise they'll go to the transaction provider.
 */
export class RoutingHandler implements FullnodeHandler {
  private readonly readOnlyClient: SimpleClient
  private readonly transactionClient: SimpleClient

  private readonly accountRateLimiter: AccountRateLimiter

  constructor(
    transactionHandlerUrl: string,
    readonlyHandlerUrl: string,
    maxRequestsPerTimeUnit: number,
    maxTransactionsPerTimeUnit: number,
    requestLimitPeriodInMillis: number,
    private readonly deployAddress: Address,
    private readonly toAddressWhitelist: Address[] = [],
    private readonly whitelistedMethods: Set<string> = new Set<string>(
      web3RpcMethodsExcludingTest
    )
  ) {
    this.readOnlyClient = new SimpleClient(readonlyHandlerUrl)
    this.transactionClient = new SimpleClient(transactionHandlerUrl)

    if (
      !!maxTransactionsPerTimeUnit &&
      !!maxTransactionsPerTimeUnit &&
      !!requestLimitPeriodInMillis
    ) {
      this.accountRateLimiter = new AccountRateLimiter(
        maxRequestsPerTimeUnit,
        maxTransactionsPerTimeUnit,
        requestLimitPeriodInMillis
      )
    }
  }

  /**
   * Handles the provided request by
   * * Checking rate limits (and throwing if there's a violation)
   * * Making sure that the destination address is allowed
   * * Routing the request to the appropriate provider.
   *
   * @param method The Ethereum JSON RPC method.
   * @param params The parameters.
   * @param sourceIpAddress The requesting IP address.
   */
  public async handleRequest(
    method: string,
    params: any[],
    sourceIpAddress: string
  ): Promise<string> {
    log.debug(
      `Proxying request for method [${method}], params: [${JSON.stringify(
        params
      )}]`
    )

    let tx: Transaction
    if (method === Web3RpcMethods.sendRawTransaction) {
      try {
        tx = parseTransaction(params[0])
      } catch (e) {
        // means improper format -- since we can't get address, add to quota by IP
        this.validateRateLimit(undefined, sourceIpAddress)
        throw new InvalidParametersError()
      }

      this.validateRateLimit(tx.from)
    } else {
      this.validateRateLimit(undefined, sourceIpAddress)
    }

    if (!this.whitelistedMethods.has(method)) {
      log.debug(
        `Received request for unsupported method: [${method}]. Supported methods: [${JSON.stringify(
          this.whitelistedMethods
        )}]`
      )
      throw new UnsupportedMethodError()
    }

    this.assertDestinationValid(tx)

    try {
      const result: any =
        methodsToRouteWithTransactionHandler.indexOf(method) >= 0
          ? await this.transactionClient.handle<string>(method, params)
          : await this.readOnlyClient.handle<string>(method, params)
      log.debug(
        `Request for [${method}], params: [${JSON.stringify(
          params
        )}] got result [${result}]`
      )
      return result
    } catch (e) {
      logError(
        log,
        `Error proxying request: [${method}], params: [${JSON.stringify(
          params
        )}]`,
        e
      )
      throw e
    }
  }

  /**
   * Validates the request is under the rate limit for the provided tx from address or
   * source IP address if a rate limit is configured
   *
   * @param txFromAddress The from address only provided if this is to check against the tx rate limit.
   * @param sourceIpAddress The IP address only provided if this is to check against the IP rate limit.
   * @throws AccountRateLimiter If the IP in question is above its rate limit.
   * @throws TransactionLimitError If the address in question is above its rate limit.
   */
  private validateRateLimit(
    txFromAddress?: string,
    sourceIpAddress?: string
  ): void {
    if (!this.accountRateLimiter) {
      return
    }
    if (!!txFromAddress) {
      this.accountRateLimiter.validateTransactionRateLimit(txFromAddress)
    } else {
      this.accountRateLimiter.validateRateLimit(sourceIpAddress)
    }
  }

  /**
   * If provided a transaction, and a transaction destination whitelist is configured,
   * this will make sure the destination of the transaction is on the whitelist or
   * the transaction is sent by the deployer address.
   *
   * @param tx The transaction in question.
   * @throws InvalidTransactionDesinationError if the transaction destination is invalid.
   */
  private assertDestinationValid(tx?: Transaction): void {
    if (
      !!tx &&
      !!this.toAddressWhitelist.length &&
      !(tx.to in this.toAddressWhitelist) &&
      tx.from !== this.deployAddress
    ) {
      throw new InvalidTransactionDesinationError(
        tx.to,
        this.toAddressWhitelist
      )
    }
  }
}
