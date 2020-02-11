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
import { Aggregator } from '../types'

const log: Logger = getLogger('aggregator-rpc-server')

/**
 * JSON RPC Server customized to Aggregator-supported methods.
 */
export class AggregatorRpcServer extends ExpressHttpServer {
  private readonly aggregator: Aggregator

  /**
   * Creates the Aggregator server
   *
   * @param supportedMethods The JSON RPC methods supported by this server
   * @param aggregator The aggregator that will fulfill requests
   * @param port Port to listen on.
   * @param hostname Hostname to listen on.
   * @param middleware any express middle
   */
  constructor(
    private readonly supportedMethods: Set<string>,
    aggregator: Aggregator,
    hostname: string,
    port: number,
    middleware?: any[]
  ) {
    super(port, hostname, middleware)
    this.aggregator = aggregator
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
        return res.json(await this.aggregator.handleRequest(request))
      } catch (err) {
        logError(log, `Uncaught exception at endpoint-level`, err)
        return res.json(buildJsonRpcError('INTERNAL_ERROR', request.id))
      }
    })
  }
}
