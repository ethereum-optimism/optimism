import { RpcTransportAdapter } from '../../adapter.interface'
import { JsonRpcRequest, JsonRpcResponse } from './json-rpc-message.interface'

export type JsonRpcAdapter<
  TransportRequest,
  TransportResponse
> = RpcTransportAdapter<
  JsonRpcRequest,
  JsonRpcResponse,
  TransportRequest,
  TransportResponse
>
