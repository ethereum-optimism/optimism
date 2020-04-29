/**
 * Wrapper class around a Http-based JsonRpcClient
 */
import { JsonRpcClient } from './json-rpc-client'
import { HttpRequest, HttpResponse, JsonRpcResponse } from '../../../types'
import { JsonRpcHttpAdapter } from './json-rpc-http-adapter'
import { AxiosHttpClient } from './axios-http-client'

export class SimpleClient {
  private jsonRpcClient: JsonRpcClient<HttpRequest, HttpResponse>

  /**
   * Initializes an internal jsonRpcClient which will be used for handling requests
   * @param baseUrl the url which json RPC requests will be sent to
   */
  constructor(private baseUrl: string) {
    this.jsonRpcClient = new JsonRpcClient<HttpRequest, HttpResponse>(
      new JsonRpcHttpAdapter(),
      new AxiosHttpClient(baseUrl)
    )
  }

  /**
   * Handles a method call by making a JSON-RPC
   * request to some server.
   * @param method Name of the method to call.
   * @param [params] Parameters to send with the method call.
   * @returns the `result` field of the response to the method call.
   * @throws Error if there is any error, including a properly-formatted JsonRpcResponse error.
   */
  public async handle<T>(method: string, params?: any): Promise<T> {
    return this.jsonRpcClient.handle<T>(method, params)
  }

  /**
   * Makes an RPC call and returns the full JsonRpcResponse.
   * Notably, this differs from handle<T>(...) because it does not throw one error
   * or just return the `result` field on success.
   * @param method Name of the method to call.
   * @param [params] Parameters to send with the method call.
   * @returns the result of the method call.
   */
  public async makeRpcCall(
    method: string,
    params?: any
  ): Promise<JsonRpcResponse> {
    return this.jsonRpcClient.makeRpcCall(method, params)
  }
}
