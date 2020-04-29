/* External Imports */
import {
  buildJsonRpcError,
  getLogger,
  isJsonRpcRequest,
  logError,
  ExpressHttpServer,
  Logger,
  JsonRpcRequest,
  JsonRpcErrorResponse,
  JsonRpcResponse,
  JSONRPC_ERRORS,
} from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  FullnodeHandler,
  InvalidParametersError,
  InvalidTransactionDesinationError,
  RateLimitError,
  RevertError,
  TransactionLimitError,
  UnsupportedMethodError,
  UnsupportedFilterError
} from '../types'

const log: Logger = getLogger('rollup-fullnode-rpc-server')

/**
 * JSON RPC Server customized to Rollup Fullnode-supported methods.
 */
export class FullnodeRpcServer extends ExpressHttpServer {
  private readonly fullnodeHandler: FullnodeHandler

  /**
   * Creates the Fullnode RPC server
   *
   * @param fullnodeHandler The fullnodeHandler that will fulfill requests
   * @param port Port to listen on.
   * @param hostname Hostname to listen on.
   * @param middleware any express middle
   */
  constructor(
    fullnodeHandler: FullnodeHandler,
    hostname: string,
    port: number,
    middleware?: any[]
  ) {
    super(port, hostname, middleware)
    this.fullnodeHandler = fullnodeHandler
  }

  /**
   * Initializes app routes.
   */
  protected initRoutes(): void {
    this.app.post('/', async (req, res) => {
      return res.json(await this.handleRequest(req))
    })
  }

  /**
   * Handles the provided request, returning the appropriate response object
   * @param req The request to handle
   * @param inBatchRequest Whether or not this is being called within the context of handling a batch requset
   * @returns The JSON-stringifiable response object.
   */
  protected async handleRequest(
    req: any,
    inBatchRequest: boolean = false
  ): Promise<JsonRpcResponse | JsonRpcResponse[]> {
    let request: JsonRpcRequest

    try {
      request = req.body
      if (Array.isArray(request) && !inBatchRequest) {
        log.debug(`Received batch request: [${JSON.stringify(request)}]`)
        const requestArray: any[] = request as any[]
        return Promise.all(
          requestArray.map(
            (x) =>
              this.handleRequest({ body: x }, true) as Promise<JsonRpcResponse>
          )
        )
      }

      if (!isJsonRpcRequest(request)) {
        log.debug(
          `Received request of unsupported format: [${JSON.stringify(request)}]`
        )
        return buildJsonRpcError('INVALID_REQUEST', null)
      }

      const sourceIpAddress = FullnodeRpcServer.getIpAddressFromRequest(req)
      const result = await this.fullnodeHandler.handleRequest(
        request.method,
        request.params,
        sourceIpAddress
      )
      return {
        id: request.id,
        jsonrpc: request.jsonrpc,
        result,
      }
    } catch (err) {
      if (err instanceof RevertError) {
        log.debug(`Request reverted. Request: ${JSON.stringify(request)}`)
        const errorResponse: JsonRpcErrorResponse = buildJsonRpcError(
          'REVERT_ERROR',
          request.id
        )
        errorResponse.error.message = err.message
        return errorResponse
      }
      if (err instanceof UnsupportedMethodError) {
        log.debug(
          `Received request with unsupported method: [${JSON.stringify(
            request
          )}]`
        )
        return buildJsonRpcError('METHOD_NOT_FOUND', request.id)
      }
      if (err instanceof UnsupportedFilterError) {
        log.debug(
          `Received request with unsupported filter parameters: [${JSON.stringify(
            request
          )}]`
        )
        return buildJsonRpcError('UNSUPPORTED_TOPICS_ERROR', request.id)
      }
      if (err instanceof InvalidParametersError) {
        log.debug(
          `Received request with valid method but invalid parameters: [${JSON.stringify(
            request
          )}]`
        )
        return buildJsonRpcError('INVALID_PARAMS', request.id)
      }
      if (err instanceof InvalidTransactionDesinationError) {
        const destErr = err as InvalidTransactionDesinationError
        log.debug(
          `Received tx request to an invalid destination [${
            destErr.destinationAddress
          }]. Valid destinations: [${JSON.stringify(
            destErr.validDestinationAddresses
          )}]. Request: [${JSON.stringify(request)}]`
        )
        return buildJsonRpcError('INVALID_PARAMS', request.id)
      }
      if (err instanceof RateLimitError) {
        const rateLimitError = err as RateLimitError
        const msg = `Request puts ${rateLimitError.ipAddress}} over limit of ${rateLimitError.limitPerPeriod}} requests every ${rateLimitError.periodInMillis}ms. Total this period: ${rateLimitError.requestCount}}.`
        log.debug(`${msg} Request: [${JSON.stringify(request)}]`)
        return {
          jsonrpc: '2.0',
          error: {
            code: -32005,
            message: msg,
          },
          id: request.id,
        }
      }
      if (err instanceof TransactionLimitError) {
        const txLimitError = err as TransactionLimitError
        const msg = `Request puts ${txLimitError.address}} over limit of ${txLimitError.limitPerPeriod}} requests every ${txLimitError.periodInMillis}ms. Total this period: ${txLimitError.transactionCount}}.`
        log.debug(`${msg} Request: [${JSON.stringify(request)}]`)
        return {
          jsonrpc: '2.0',
          error: {
            code: -32005,
            message: msg,
          },
          id: request.id,
        }
      }

      logError(log, `Uncaught exception at endpoint-level`, err)
      return buildJsonRpcError(
        'INTERNAL_ERROR',
        request && request.id ? request.id : null
      )
    }
  }

  private static getIpAddressFromRequest(req: any): string {
    if (!!req.ip) {
      return req.ip
    }
    if (!!req.headers && !!req.headers['x-forwarded-for']) {
      return req.headers['x-forwarded-for']
    }
    if (!!req.connection && !!req.connection.remoteAddress) {
      return req.connection.remoteAddress
    }
    return undefined
  }
}
