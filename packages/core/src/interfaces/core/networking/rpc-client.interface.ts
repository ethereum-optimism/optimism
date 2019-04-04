/**
 * RpcClient exposes an interface for interacting with other nodes.
 */
export interface RpcClient {
  /**
   * Handles an RPC request.
   * @param method to call.
   * @param params to call the method with.
   * @returns the result of the RPC request.
   */
  handle<T>(method: string, params?: any[]): Promise<T>
}
