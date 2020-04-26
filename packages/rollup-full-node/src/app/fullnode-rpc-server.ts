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
} from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  FullnodeHandler,
  InvalidParametersError,
  RevertError,
  UnsupportedMethodError,
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

      const result = await this.fullnodeHandler.handleRequest(
        request.method,
        request.params
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
      } else if (err instanceof InvalidParametersError) {
        log.debug(
          `Received request with valid method but invalid parameters: [${JSON.stringify(
            request
          )}]`
        )
        return buildJsonRpcError('INVALID_PARAMS', request.id)
      }
      logError(log, `Uncaught exception at endpoint-level for request [${JSON.stringify(request)}]:`, err)
      return buildJsonRpcError(
        'INTERNAL_ERROR',
        request && request.id ? request.id : null
      )
    }
  }
}
