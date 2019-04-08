/* External Imports */
import axios, { AxiosRequestConfig, AxiosResponse } from 'axios'

/* Internal Imports */
import { HttpClient } from '../../../../../interfaces'

/**
 * HTTP client that uses the axios client library.
 */
export class AxiosHttpClient implements HttpClient {
  private http = axios.create({
    baseURL: this.baseUrl,
  })

  /**
   * Creates the client.
   * @param baseUrl Base URL to make requests to.
   */
  constructor(private baseUrl: string) {}

  /**
   * Sends an HTTP request to a server.
   * @param data Data to send to the server.
   * @returns the HTTP response.
   */
  public async request(data: AxiosRequestConfig): Promise<AxiosResponse> {
    return this.http.request(data)
  }
}
