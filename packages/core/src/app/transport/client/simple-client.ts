import { JsonRpcClient, JsonRpcHttpAdapter, AxiosHttpClient } from '../../../app'
import { HttpRequest, HttpResponse } from '../../../types'

/**
 * Wrapper class around a Http-based JsonRpcClient
 */
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
   * @returns the result of the method call.
   */
  public async handle<T>(method: string, params?: any): Promise<T> {
    return this.jsonRpcClient.handle<T>(method, params)
  }
}
