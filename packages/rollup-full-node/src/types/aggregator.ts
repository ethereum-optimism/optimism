/** External Imports */
import { JsonRpcRequest, JsonRpcResponse } from '@eth-optimism/core-utils'

export interface Aggregator {
  handleRequest(request: JsonRpcRequest): Promise<JsonRpcResponse>
}
