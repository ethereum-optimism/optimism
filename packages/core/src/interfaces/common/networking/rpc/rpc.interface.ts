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

/**
 * Handles responses to an RpcClient.
 */
export interface RpcServer {
  /**
   * Registers a method so it can be exposed.
   * @param name Name of the method.
   * @param method Function to call.
   */
  register(name: string, method: Function): void

  /**
   * Starts the server.
   */
  listen(): Promise<void>
}
