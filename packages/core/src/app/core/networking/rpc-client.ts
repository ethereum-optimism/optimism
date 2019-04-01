import { RpcClient } from '../../../interfaces'

export class DefaultRpcClient implements RpcClient {
  public async handle<T>(method: string, params?: any): Promise<T> {

  }
}
