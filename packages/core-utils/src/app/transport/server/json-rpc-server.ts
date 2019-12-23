/* Internal Imports */
import {
  JsonRpcRequest,
  JsonRpcSuccessResponse,
  RpcServer,
  isJsonRpcRequest,
} from '../../../types'
import { buildJsonRpcError } from './json-rpc-errors'
import { ExpressHttpServer } from './express-http-server'

/**
 * Basic JSON-RPC server.
 */
export class JsonRpcServer extends ExpressHttpServer implements RpcServer {
  /**
   * Creates the server
   * @param methods Methods to expose to the server.
   * @param port Port to listen on.
   * @param hostname Hostname to listen on.
   */
  constructor(
    private methods: Record<string, Function> = {},
    hostname: string,
    port: number,
    middleware?: any[]
  ) {
    super(port, hostname, middleware)
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

      if (!(request.method in this.methods)) {
        return res.json(buildJsonRpcError('METHOD_NOT_FOUND', request.id))
      }

      let result: any
      try {
        result = await this.methods[request.method](request.params)
      } catch (err) {
        return res.json(buildJsonRpcError('INTERNAL_ERROR', request.id))
      }

      const response: JsonRpcSuccessResponse = {
        jsonrpc: request.jsonrpc,
        id: request.id,
        result,
      }
      return res.json(response)
    })
  }

  /**
   * Registers a method so the server can expose it.
   * @param name Name of the method to expose.
   * @param method Function to run.
   */
  public register(name: string, method: Function): void {
    if (name in this.methods) {
      throw new Error(`method already registered: ${name}`)
    }

    this.methods[name] = method
  }
}
