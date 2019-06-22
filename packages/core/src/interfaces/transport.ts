/*************** adapter ***************/
/**
 * Handles conversion between RPC format and transport layer format.
 */
export interface RpcTransportAdapter<
  RpcRequest,
  RpcResponse,
  TransportRequest,
  TransportResponse
> {
  /**
   * Encodes an RPC request into a request
   * understood by the transport layer.
   * @param request RPC request to encode.
   * @returns the encoded request.
   */
  encodeRequest(request: RpcRequest): TransportRequest

  /**
   * Decodes a response from the transport layer
   * into a request understood by the RPC protocol.
   * @param response Transport layer response to encode.
   * @returns the decoded response.
   */
  decodeResponse(response: TransportResponse): RpcResponse
}


/*************** rpc ***************/
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


/*************** json-rpc-adapter ***************/
export type JsonRpcAdapter<
  TransportRequest,
  TransportResponse
> = RpcTransportAdapter<
  JsonRpcRequest,
  JsonRpcResponse,
  TransportRequest,
  TransportResponse
>


/*************** json-rpc-message ***************/
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


/*************** transport ***************/
/**
 * Client sends some data to a server over a transport layer.
 */
export interface Client<Request, Response> {
  /**
   * Sends a request over some transport layer.
   * @param data Data to send.
   * @returns the response of the request.
   */
  request(data: Request): Promise<Response>
}

/**
 * Server sends some data to a client over a transport layer.
 */
export interface Server {
  /**
   * Starts the server.
   */
  listen(): void
}


/*************** transport-http-message ***************/
export interface HttpRequest {
  url: string
  method:
    | 'get'
    | 'GET'
    | 'head'
    | 'HEAD'
    | 'post'
    | 'POST'
    | 'put'
    | 'PUT'
    | 'delete'
    | 'DELETE'
    | 'options'
    | 'OPTIONS'
    | 'patch'
    | 'PATCH'
  headers?: Record<any, any>
  params?: Record<any, any>
  data?: any
  timeout?: number
}

export interface HttpResponse {
  status: number
  statusText: string
  headers?: Record<any, any>
  data?: any
}


/*************** transport-http-message ***************/
export type HttpClient = Client<HttpRequest, HttpResponse>
export type HttpServer = Server
