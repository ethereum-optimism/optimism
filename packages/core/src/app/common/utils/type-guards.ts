import { JsonRpcResponse, JsonRpcErrorResponse } from '../../../interfaces'

export const isJsonRpcErrorResponse = (
  response: JsonRpcResponse
): response is JsonRpcErrorResponse => {
  return typeof (response as JsonRpcErrorResponse).error !== 'undefined'
}
