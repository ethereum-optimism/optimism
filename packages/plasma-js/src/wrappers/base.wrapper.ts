import { BaseProvider, HttpProvider } from '../providers'

export class BaseWrapper {
  protected prefix = ''
  protected provider: BaseProvider

  constructor(provider: string | BaseProvider = new HttpProvider()) {
    if (typeof provider === 'string') {
      provider = new HttpProvider({
        endpoint: provider,
      })
    }
    this.provider = provider
  }

  protected async handle(method: string, params?: any[]): Promise<any> {
    return this.provider.handle(`${this.prefix}_${method}`, params)
  }
}
