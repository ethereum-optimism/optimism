/* External Imports */
import uuid = require('uuid')

/* Internal Imports */
import {
  RpcClient,
  JsonRpcAdapter,
  JsonRpcRequest,
  Client,
  isJsonRpcErrorResponse,
  JsonRpcResponse,
} from '../../types'

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
   *
   * @param method Name of the method to call.
   * @param [params] Parameters to send with the method call.
   * @returns the `result` field of the response to the method call.
   * @throws Error if there is any error, including a properly-formatted JsonRpcResponse error.
   */
  public async handle<T>(method: string, params?: any): Promise<T> {
    const response: JsonRpcResponse = await this.makeRpcCall(method, params)

    if (isJsonRpcErrorResponse(response)) {
      throw new Error(`${JSON.stringify(response.error)}`)
    }

    return response.result
  }

  /**
   * Makes an RPC call and returns the full JsonRpcResponse.
   * Notably, this differs from handle<T>(...) because it does not throw on error
   * or just return the `result` field on success.
   *
   * @param method Name of the method to call.
   * @param [params] Parameters to send with the method call.
   * @returns the result of the method call.
   */
  public async makeRpcCall(
    method: string,
    params?: any
  ): Promise<JsonRpcResponse> {
    const request: JsonRpcRequest = {
      jsonrpc: '2.0',
      method,
      params,
      id: uuid.v4(),
    }

    const encodedRequest = this.adapter.encodeRequest(request)
    const encodedResponse = await this.client.request(encodedRequest)
    return this.adapter.decodeResponse(encodedResponse)
  }
}
