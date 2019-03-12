export type JSONRPCParam = string | number
export type JSONRPCResult = string | number | {}

export interface JSONRPCError {
  code: number
  message: string
}

export interface JSONRPCRequest {
  jsonrpc: string
  method: string
  id: string
  params: JSONRPCParam[]
}

export interface JSONRPCResponse {
  jsonrpc: string
  result?: JSONRPCResult
  error?: JSONRPCError
  message?: string
  id: string | null
}
