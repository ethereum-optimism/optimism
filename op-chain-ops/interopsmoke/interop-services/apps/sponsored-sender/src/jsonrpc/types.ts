/**
 * The error object in a JSON RPC response.
 */
export type JsonRpcError = {
  error: {
    code: number
    message: string
  }
}

/**
 * The result object in a JSON RPC response.
 */
export type JsonRpcResult = {
  result: unknown
}

/**
 * The response object in a JSON RPC response.
 */
export type JsonRpcResponse = {
  jsonrpc: '2.0'
  id: number | string | null
} & (JsonRpcError | JsonRpcResult)

/**
 * The request object in a JSON RPC request.
 */
export type JsonRpcRequest = {
  jsonrpc: '2.0'
  id: number | string
  method: string
  params?: unknown
}
