/**
 * RpcClient handles requests to an RpcServer.
 */
export interface RpcClient {
  /**
   * Handles some RPC requests.
   * @param method Name of the method to call.
   * @param [params] Extra parameters to send to the call.
   * @returns the RPC response.
   */
  handle<T>(method: string, params?: any): Promise<T>
}
