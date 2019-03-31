export interface JsonRpcMessage {
  jsonrpc: '2.0'
  id: string | number | null
}

export interface JsonRpcRequest extends JsonRpcMessage {
  method: string
  params?: any[]
}

export interface JsonRpcError {
  code: number
  message: string
  data?: any
}

export interface JsonRpcErrorResponse extends JsonRpcMessage {
  error: JsonRpcError
}

export interface JsonRpcSuccessResponse extends JsonRpcMessage {
  result: any
}

export type JsonRpcResponse = JsonRpcSuccessResponse | JsonRpcErrorResponse
