/* External Imports */
import { Address } from '@eth-optimism/rollup-core'

import { JsonRpcProvider } from 'ethers/providers'
import { parseTransaction, Transaction } from 'ethers/utils'

/* Internal Imports */
import {
  FullnodeHandler,
  InvalidParametersError,
  InvalidTransactionDesinationError,
} from '../../types'
import { AccountRateLimiter } from '../utils'

const sendRawTransactionMethod = 'sendRawTransaction'
const methodsToRouteWithTransactionHandler: string[] = [
  sendRawTransactionMethod,
  'getOvmTransactionByHash',
  'getBlockByNumber',
  'getBlockByHash',
]

/**
 * Handles rate-limiting requests by Ethereum address for transactions and by IP address for all
 * other request types and then routes them according to their method.
 *
 * If they are read-only they'll go to the provided read-only provider, and
 * otherwise they'll go to the transaction provider.
 */
export class RoutingHandler implements FullnodeHandler {
  private readonly readonlyProvider: JsonRpcProvider
  private readonly transactionProvider: JsonRpcProvider

  private readonly accountRateLimiter: AccountRateLimiter

  constructor(
    transactionHandlerUrl: string,
    readonlyHandlerUrl: string,
    maxRequestsPerTimeUnit: number,
    maxTransactionsPerTimeUnit: number,
    requestLimitPeriodInMillis: number,
    private readonly deployAddress: Address,
    private readonly toAddressWhitelist?: Address[]
  ) {
    this.readonlyProvider = new JsonRpcProvider(readonlyHandlerUrl)
    this.transactionProvider = new JsonRpcProvider(transactionHandlerUrl)

    this.accountRateLimiter = new AccountRateLimiter(
      maxRequestsPerTimeUnit,
      maxTransactionsPerTimeUnit,
      requestLimitPeriodInMillis
    )
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
    let tx: Transaction

    if (method === sendRawTransactionMethod) {
      try {
        tx = parseTransaction(params[0])
      } catch (e) {
        // means improper format -- since we can't get address, add to quota by IP
        this.accountRateLimiter.validateRateLimit(sourceIpAddress)
        throw new InvalidParametersError()
      }

      this.accountRateLimiter.validateTransactionRateLimit(tx.from)
    } else {
      this.accountRateLimiter.validateRateLimit(sourceIpAddress)
    }

    if (
      !!tx &&
      !!this.toAddressWhitelist &&
      !(tx.to in this.toAddressWhitelist) &&
      tx.from !== this.deployAddress
    ) {
      throw new InvalidTransactionDesinationError(
        tx.to,
        this.toAddressWhitelist
      )
    }

    if (method in methodsToRouteWithTransactionHandler) {
      return JSON.stringify(await this.transactionProvider.send(method, params))
    } else {
      return JSON.stringify(await this.readonlyProvider.send(method, params))
    }
  }
}
