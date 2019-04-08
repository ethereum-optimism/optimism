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
