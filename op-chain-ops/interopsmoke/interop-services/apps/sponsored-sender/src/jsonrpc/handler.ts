import type { z } from 'zod'
import { fromZodError } from 'zod-validation-error'

import type { JsonRpcRequest, JsonRpcResponse } from '@/jsonrpc/types.js'

/**
 * A JSON RPC Handler to handle JSON RPC requests and return responses.
 */
export class JsonRpcHandler {
  private methodByName: Record<
    string,
    {
      paramsValidator: z.ZodTypeAny
      handler: Function
    }
  > = {}

  /**
   * Registers a new handler for a JSON RPC method.
   * @param methodName - The name of the method to register.
   * @param paramsValidator - The ZOD schema for the parameters of the method.
   * @param handler - The handler for the method.
   */
  method<T extends z.ZodTypeAny, U>(
    methodName: string,
    paramsValidator: T,
    handler: (params: z.infer<T>) => Promise<U>,
  ) {
    this.methodByName[methodName] = {
      paramsValidator,
      handler,
    }
  }

  /**
   * Handles a JSON RPC request.
   * @param request - The JSON RPC request or an array of JSON RPC requests.
   * @returns A JSON RPC response or an array of JSON RPC responses.
   */
  async handle(
    request: JsonRpcRequest | JsonRpcRequest[],
  ): Promise<JsonRpcResponse | JsonRpcResponse[]> {
    if (Array.isArray(request) && request.length === 0) {
      return {
        jsonrpc: '2.0',
        id: null,
        error: { code: -32600, message: 'Invalid request' },
      }
    }

    if (Array.isArray(request)) {
      return Promise.all(request.map(this.__handleSingle.bind(this)))
    }

    return this.__handleSingle(request)
  }

  private async __handleSingle(
    request: JsonRpcRequest,
  ): Promise<JsonRpcResponse> {
    const { method, params, id } = request
    if (!this.methodByName[method]) {
      return {
        jsonrpc: '2.0',
        id,
        error: { code: -32601, message: 'Method not found' },
      }
    }

    const { handler, paramsValidator } = this.methodByName[method]
    const { data, error, success } = paramsValidator.safeParse(params)
    if (!success) {
      const { message } = fromZodError(error, { maxIssuesInMessage: 1 })
      return {
        jsonrpc: '2.0',
        id,
        error: { code: -32602, message: `Invalid params: ${message}` },
      }
    }

    return handler(data)
      .then((result: unknown) => {
        return { jsonrpc: '2.0', id, result }
      })
      .catch((e: unknown) => {
        return {
          jsonrpc: '2.0',
          id,
          error: {
            code: -32603,
            message: `Proxy error: ${
              e instanceof Error ? e.message : JSON.stringify(e)
            }`,
          },
        }
      })
  }
}
