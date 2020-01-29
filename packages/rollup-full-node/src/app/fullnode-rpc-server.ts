/* External Imports */
import {
  buildJsonRpcError,
  getLogger,
  isJsonRpcRequest,
  logError,
  ExpressHttpServer,
  Logger,
  JsonRpcRequest,
} from '@pigi/core-utils'
import { FullnodeHandler } from '../types'

const log: Logger = getLogger('rollup-fullnode-rpc-server')

/**
 * JSON RPC Server customized to Rollup Fullnode-supported methods.
 */
export class FullnodeRpcServer extends ExpressHttpServer {
  private readonly fullnodeHandler: FullnodeHandler

  /**
   * Creates the Fullnode RPC server
   *
   * @param supportedMethods The JSON RPC methods supported by this server
   * @param fullnodeHandler The fullnodeHandler that will fulfill requests
   * @param port Port to listen on.
   * @param hostname Hostname to listen on.
   * @param middleware any express middle
   */
  constructor(
    private readonly supportedMethods: Set<string>,
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
      const request: JsonRpcRequest = req.body
      if (!isJsonRpcRequest(request)) {
        return res.json(buildJsonRpcError('INVALID_REQUEST', null))
      }

      if (!this.supportedMethods.has(request.method)) {
        return res.json(buildJsonRpcError('METHOD_NOT_FOUND', request.id))
      }

      try {
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
        logError(log, `Uncaught exception at endpoint-level`, err)
        return res.json(buildJsonRpcError('INTERNAL_ERROR', request.id))
      }
    })
  }
}
