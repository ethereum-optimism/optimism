import express = require('express')
import bodyParser = require('body-parser')

import { JsonRpcRequest, JsonRpcSuccessResponse, RpcServer } from '../../../interfaces'
import { buildJsonRpcError, isJsonRpcRequest } from '../../common'

/**
 * Basic JSON-RPC server.
 */
export class SimpleJsonRpcServer implements RpcServer {
  private app = express()
  private listening = false

  /**
   * Creates the server
   * @param methods Methods to expose to the server.
   * @param port Port to listen on.
   * @param hostname Hostname to listen on.
   */
  constructor(
    private methods: Record<string, Function> = {},
    private port: number,
    private hostname: string
  ) {
    this.app.use(bodyParser.json())
    this.app.get('/', async (req, res) => {
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
      } catch {
        return res.json(buildJsonRpcError('INTERNAL_ERROR', request.id))
      }

      const response: JsonRpcSuccessResponse = {
        jsonrpc: '2.0',
        id: request.id,
        result,
      }
      return res.json(response)
    })
  }

  /**
   * Starts the server.
   */
  public async listen(): Promise<void> {
    if (this.listening) {
      return
    }

    return new Promise<void>((resolve, reject) => {
      this.app.listen(this.port, this.hostname, () => {
        this.listening = true
        resolve()
      })
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
