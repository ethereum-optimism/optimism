/* External Imports */
import { Service } from '@nestd/core'

/* Services */
import { LoggerService } from '../logger.service'

/* Internal Imports */
import { JsonRpcParam, JsonRpcRequest, JsonRpcResponse } from '../../models/rpc'
import { JSONRPC_ERRORS } from './errors'
import { BaseRpcModule } from './rpc-modules/base-rpc-module'
import {
  OperatorRpcModule,
  ChainRpcModule,
  EthRpcModule,
  WalletRpcModule,
} from './rpc-modules'

@Service()
export class JsonRpcService {
  public rpcModules: BaseRpcModule[] = []
  private readonly name = 'jsonrpc'

  constructor(
    private readonly logger: LoggerService,
    private readonly chainRpcModule: ChainRpcModule,
    private readonly ethRpcModule: EthRpcModule,
    private readonly operatorRpcModule: OperatorRpcModule,
    private readonly walletRpcModule: WalletRpcModule
  ) {
    const rpcModules = [
      this.chainRpcModule,
      this.ethRpcModule,
      this.operatorRpcModule,
      this.walletRpcModule,
    ]
    for (const rpcModule of rpcModules) {
      this.registerRpcModule(rpcModule)
    }
  }

  /**
   * Registers an RPC module to the RPC service.
   * @param rpcModule Module to register.
   */
  public registerRpcModule(rpcModule: BaseRpcModule): void {
    this.rpcModules.push(rpcModule)
  }

  /**
   * Returns all methods of all subdispatchers.
   * @returns all subdispatcher methods as a single object.
   */
  public getAllMethods(): { [key: string]: (...args: any) => any } {
    return this.rpcModules
      .map((rpcModule) => {
        return rpcModule.getAllMethods()
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
    params: JsonRpcParam[] = []
  ): Promise<string | number | {}> {
    const fn = this.getMethod(method)
    return fn(...params)
  }

  /**
   * Handles a raw (JSON) JSON-RPC request.
   * @param request A JSON-RPC request object.
   * @return the result of the JSON-RPC call.
   */
  public async handleRawRequest(
    request: JsonRpcRequest
  ): Promise<JsonRpcResponse> {
    if (!('method' in request && 'id' in request)) {
      return this.buildError('INVALID_REQUEST', null)
    }

    try {
      this.getMethod(request.method)
    } catch (err) {
      this.logger.error(this.name, 'Could not find JSON-RPC method', err)
      return this.buildError('METHOD_NOT_FOUND', request.id, err)
    }

    let result: any
    try {
      result = await this.handle(request.method, request.params)
    } catch (err) {
      this.logger.error(this.name, 'Internal JSON-RPC error', err)
      return this.buildError('INTERNAL_ERROR', request.id, err)
    }

    return {
      jsonrpc: '2.0',
      result,
      id: request.id,
    }
  }

  /**
   * Builds a JSON-RPC error response.
   * @param type Error type.
   * @param id RPC command ID.
   * @param err An error message.
   * @returns a stringified JSON-RPC error response.
   */
  private buildError(
    type: string,
    id: string | null,
    message?: string
  ): JsonRpcResponse {
    return {
      error: JSONRPC_ERRORS[type],
      id,
      jsonrpc: '2.0',
      message,
    }
  }
}
