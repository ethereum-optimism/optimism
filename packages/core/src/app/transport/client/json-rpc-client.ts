/* External Imports */
import uuid = require('uuid')

/* Internal Imports */
import {
  RpcClient,
  JsonRpcAdapter,
  JsonRpcRequest,
  Client,
  isJsonRpcErrorResponse,
} from '../../../types'

/**
 * Client for making requests to a JSON-RPC server.
 */
export class JsonRpcClient<TransportRequest, TransportResponse>
  implements RpcClient {
  constructor(
    private adapter: JsonRpcAdapter<TransportRequest, TransportResponse>,
    private client: Client<TransportRequest, TransportResponse>
  ) {}

  /**
   * Handles a method call by making a JSON-RPC
   * request to some server.
   * @param method Name of the method to call.
   * @param [params] Parameters to send with the method call.
   * @returns the result of the method call.
   */
  public async handle<T>(method: string, params?: any): Promise<T> {
    const request: JsonRpcRequest = {
      jsonrpc: '2.0',
      method,
      params,
      id: uuid.v4(),
    }

    const encodedRequest = this.adapter.encodeRequest(request)
    const encodedResponse = await this.client.request(encodedRequest)
    const response = this.adapter.decodeResponse(encodedResponse)

    if (isJsonRpcErrorResponse(response)) {
      throw new Error(`${JSON.stringify(response.error)}`)
    }
    return response.result
  }
}
