/* External Imports */
import {
  buildJsonRpcError,
  getLogger,
  isJsonRpcRequest,
  logError,
  ExpressHttpServer,
  Logger,
  JsonRpcRequest,
} from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  FullnodeHandler,
  InvalidParametersError,
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
      let request: JsonRpcRequest
      try {
        request = req.body
        if (!isJsonRpcRequest(request)) {
          log.debug(`Received request of unsupported format: [${request}]`)
          return res.json(buildJsonRpcError('INVALID_REQUEST', null))
        }

        const result = await this.fullnodeHandler.handleRequest(
          request.method,
          request.params
        )
        return res.json({
          id: request.id,
          jsonrpc: request.jsonrpc,
          result,
        })
      } catch (err) {
        if (err instanceof UnsupportedMethodError) {
          log.debug(
            `Received request with unsupported method: [${JSON.stringify(
              request
            )}]`
          )
          return res.json(buildJsonRpcError('METHOD_NOT_FOUND', request.id))
        } else if (err instanceof InvalidParametersError) {
          log.debug(
            `Received request with valid method but invalid parameters: [${JSON.stringify(
              request
            )}]`
          )
          return res.json(buildJsonRpcError('INVALID_PARAMS', request.id))
        }
        logError(log, `Uncaught exception at endpoint-level`, err)
        return res.json(buildJsonRpcError('INTERNAL_ERROR', request.id))
      }
    })
  }
}
