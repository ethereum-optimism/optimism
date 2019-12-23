/** External Imports */
import { JsonRpcRequest, JsonRpcResponse } from '@pigi/core-utils'

export interface Aggregator {
  handleRequest(request: JsonRpcRequest): Promise<JsonRpcResponse>
}
