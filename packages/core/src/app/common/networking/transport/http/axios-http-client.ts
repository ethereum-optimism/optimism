import axios, { AxiosRequestConfig, AxiosResponse } from 'axios'

import { HttpClient } from '../../../../../interfaces'

/**
 * HTTP client that uses the axios client library.
 */
export class AxiosHttpClient implements HttpClient {
  private http = axios.create({
    baseURL: this.baseUrl,
  })

  constructor(private baseUrl: string) {}

  /**
   * Sends an HTTP request to a server.
   * @param data
   */
  async request(data: AxiosRequestConfig): Promise<AxiosResponse> {
    return this.http.request(data)
  }
}
