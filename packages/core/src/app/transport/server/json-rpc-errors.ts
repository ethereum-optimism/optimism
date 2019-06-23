import { JsonRpcErrorResponse } from 'src/types'

export const JSONRPC_ERRORS = {
  PARSE_ERROR: {
    code: -32700,
    message: 'Parse error',
  },
  INVALID_REQUEST: {
    code: -32600,
    message: 'Invalid request',
  },
  METHOD_NOT_FOUND: {
    code: -32601,
    message: 'Method not found',
  },
  INVALID_PARAMS: {
    code: -32602,
    message: 'Invalid params',
  },
  INTERNAL_ERROR: {
    code: -32603,
    message: 'Internal error',
  },
}

/**
 * Utility for building an error response.
 * @param type Error type for the response.
 * @param id ID for the response.
 * @returns the response object.
 */
export const buildJsonRpcError = (
  type: keyof typeof JSONRPC_ERRORS,
  id: string | number
): JsonRpcErrorResponse => {
  return {
    jsonrpc: '2.0',
    error: JSONRPC_ERRORS[type],
    id,
  }
}
