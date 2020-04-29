/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import {
  getLogger,
  isJsonRpcErrorResponse,
  JsonRpcErrorResponse,
  JsonRpcResponse,
  logError,
  SimpleClient,
} from '@eth-optimism/core-utils'

import { parseTransaction, Transaction } from 'ethers/utils'

/* Internal Imports */
import {
  AccountRateLimiter,
  FormattedJsonRpcError,
  FullnodeHandler,
  InvalidParametersError,
  InvalidTransactionDesinationError,
  UnsupportedMethodError,
  Web3RpcMethods,
  web3RpcMethodsExcludingTest,
} from '../types'

const log = getLogger('routing-handler')

const methodsToRouteWithTransactionHandler: string[] = [
  Web3RpcMethods.sendTransaction,
  Web3RpcMethods.sendRawTransaction,
  Web3RpcMethods.getTransactionByHash,
  Web3RpcMethods.getBlockByNumber,
  Web3RpcMethods.getBlockByHash,
]
export const getMethodsToRouteWithTransactionHandler = () => {
  return Array.of(...methodsToRouteWithTransactionHandler)
}

/**
 * Handles rate-limiting requests by Ethereum address for transactions and by IP address for all
 * other request types and then routes them according to their method.
 *
 * If they are read-only they'll go to the provided read-only provider, and
 * otherwise they'll go to the transaction provider.
 */
export class RoutingHandler implements FullnodeHandler {
  constructor(
    private readonly transactionClient: SimpleClient,
    private readonly readOnlyClient: SimpleClient,
    private readonly deployAddress: Address,
    private readonly accountRateLimiter: AccountRateLimiter,
    private readonly toAddressWhitelist: Address[] = [],
    private readonly whitelistedMethods: Set<string> = new Set<string>(
      web3RpcMethodsExcludingTest
    )
  ) {}

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
        this.accountRateLimiter.validateRateLimit(sourceIpAddress)
        throw new InvalidParametersError()
      }

      this.accountRateLimiter.validateTransactionRateLimit(tx.from)
    } else {
      this.accountRateLimiter.validateRateLimit(sourceIpAddress)
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
      const result: JsonRpcResponse =
        methodsToRouteWithTransactionHandler.indexOf(method) >= 0
          ? await this.transactionClient.makeRpcCall(method, params)
          : await this.readOnlyClient.makeRpcCall(method, params)
      log.debug(
        `Request for [${method}], params: [${JSON.stringify(
          params
        )}] got result [${JSON.stringify(result)}]`
      )
      if (isJsonRpcErrorResponse(result)) {
        throw new FormattedJsonRpcError(result as JsonRpcErrorResponse)
      }
      return result.result
    } catch (e) {
      if (!(e instanceof FormattedJsonRpcError)) {
        logError(
          log,
          `Error proxying request: [${method}], params: [${JSON.stringify(
            params
          )}]`,
          e
        )
      }
      throw e
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
      this.toAddressWhitelist.indexOf(tx.to) < 0 &&
      tx.from !== this.deployAddress
    ) {
      throw new InvalidTransactionDesinationError(
        tx.to,
        this.toAddressWhitelist
      )
    }
  }
}
