import uuidv4 = require('uuid/v4')
import axios, { AxiosInstance } from 'axios'

import { BaseProvider } from './base.provider'

interface JsonRpcResponse {
  result?: string
  error?: string
  message?: string
}

interface HttpProviderOptions {
  endpoint?: string
}

export class HttpProvider implements BaseProvider {
  private http: AxiosInstance

  constructor({
    endpoint = 'http://localhost:9898',
  }: HttpProviderOptions = {}) {
    this.http = axios.create({
      baseURL: endpoint,
    })
  }

  public async handle(method: string, params?: any[]): Promise<any> {
    const response = await this.http.post('/', {
      jsonrpc: '2.0',
      method,
      params,
      id: uuidv4(),
    })

    const data: JsonRpcResponse = response.data
    if (data.error) {
      throw new Error(data.message)
    }
    return data.result
  }
}
