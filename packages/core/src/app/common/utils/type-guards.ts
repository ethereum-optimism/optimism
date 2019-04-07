import { JsonRpcResponse, JsonRpcErrorResponse, JsonRpcRequest } from '../../../interfaces'

/**
 * Checks if a JSON-RPC response is an error response.
 * @param response Response to check.
 * @returns `true` if the response has an error, `false` otherwise.
 */
export const isJsonRpcErrorResponse = (
  response: JsonRpcResponse
): response is JsonRpcErrorResponse => {
  return typeof (response as JsonRpcErrorResponse).error !== 'undefined'
}

/**
 * Checks if a request is a valid JSON-RPC request.
 * @param request Request to check.
 * @returns `true` if the request is a valid JSON-RPC request, `false` otherwise.
 */
export const isJsonRpcRequest = (request: any): request is JsonRpcRequest => {
  return (
    request.method !== undefined &&
    request.id !== undefined &&
    request.jsonrpc === '2.0'
  )
}
