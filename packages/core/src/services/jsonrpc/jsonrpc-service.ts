/* External Imports */
import { Service } from '@nestd/core'

/* Services */
import { LoggerService } from '../logger.service'

/* Internal Imports */
import { JSONRPCParam, JSONRPCRequest, JSONRPCResponse } from '../../models/rpc'
import { JSONRPC_ERRORS } from './errors'
import { BaseSubdispatcher } from './subdispatchers/base-subdispatcher'

@Service()
export class JSONRPCService {
  public subdispatchers: BaseSubdispatcher[] = []
  private readonly name = 'jsonrpc'

  constructor(private readonly logger: LoggerService) {}

  /**
   * Returns all methods of all subdispatchers.
   * @returns all subdispatcher methods as a single object.
   */
  public getAllMethods(): { [key: string]: (...args: any) => any } {
    return this.subdispatchers
      .map((subdispatcher) => {
        return subdispatcher.getAllMethods()
      })
      .reduce((pre, cur) => {
        return { ...pre, ...cur }
      })
  }

  /**
   * Returns a single method.
   * @param name Name of the method to return.
   * @returns the method with the given name or
   * `undefined` if the method does not exist.
   */
  public getMethod(name: string): (...args: any) => any {
    const methods = this.getAllMethods()
    if (name in methods) {
      return methods[name]
    }
    throw new Error('Method not found.')
  }

  /**
   * Calls the method with the given name and parameters.
   * @param method Name of the method to call.
   * @param params Parameters to be used as arguments to the method.
   * @returns the result of the function call.
   */
  public async handle(
    method: string,
    params: JSONRPCParam[] = []
  ): Promise<string | number | {}> {
    const fn = this.getMethod(method)
    return fn(...params)
  }

  /**
   * Handles a raw (JSON) JSON-RPC request.
   * @param request A JSON-RPC request object.
   * @return the result of the JSON-RPC call.
   */
  public async handleRawRequest(request: JSONRPCRequest) {
    if (!('method' in request && 'id' in request)) {
      return this.buildError('INVALID_REQUEST', null)
    }

    try {
      this.getMethod(request.method)
    } catch (err) {
      this.logger.error(this.name, 'Could not find JSON-RPC method', err)
      return this.buildError('METHOD_NOT_FOUND', request.id, err)
    }

    let result
    try {
      result = await this.handle(request.method, request.params)
    } catch (err) {
      this.logger.error(this.name, 'Internal JSON-RPC error', err)
      return this.buildError('INTERNAL_ERROR', request.id, err)
    }

    const response: JSONRPCResponse = { jsonrpc: '2.0', result, id: request.id }
    return JSON.stringify(response)
  }

  /**
   * Builds a JSON-RPC error response.
   * @param type Error type.
   * @param id RPC command ID.
   * @param err An error message.
   * @returns a stringified JSON-RPC error response.
   */
  private buildError(type: string, id: string | null, message?: string): {} {
    const error: JSONRPCResponse = {
      error: JSONRPC_ERRORS[type],
      id,
      jsonrpc: '2.0',
      message,
    }
    return JSON.stringify(error)
  }
}
