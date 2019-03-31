import { RpcServer } from '../../../interfaces'

export class DefaultRpcServer implements RpcServer {
  public async handle<T>(method: string, params?: any): Promise<T> {}
}
