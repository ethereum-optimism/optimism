import { RpcClient } from '../../../interfaces'
import {
  JsonRpcClient,
  JsonRpcHttpAdapter,
  AxiosHttpClient,
} from '../../common'
import { Service } from '@nestd/core';

/**
 * Basic RPC client that uses JSON-RPC over HTTP.
 */
@Service()
export class DefaultRpcClient implements RpcClient {
  private client = new JsonRpcClient(
    new JsonRpcHttpAdapter(),
    new AxiosHttpClient('')
  )

  public async handle<T>(method: string, params?: any): Promise<T> {
    return this.client.handle(method, params)
  }
}
