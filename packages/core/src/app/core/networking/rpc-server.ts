import { RpcServer } from '../../../interfaces'
import { Service } from '@nestd/core';

@Service()
export class DefaultRpcServer implements RpcServer {
  public async handle<T>(method: string, params?: any): Promise<T> {
    return
  }
}
