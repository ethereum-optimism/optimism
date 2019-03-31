/**
 * RpcServer exposes an interface for interacting with the app.
 */
export interface RpcServer {
  /**
   * Handles an RPC request.
   * @param method to call.
   * @param params to call the method with.
   * @returns the result of the RPC request.
   */
  handle(method: string, params?: any[]): Promise<any>
}
