export interface HttpRequest {
  url: string
  method:
    | 'get'
    | 'head'
    | 'post'
    | 'put'
    | 'delete'
    | 'connect'
    | 'options'
    | 'trace'
    | 'patch'
  headers?: Record<any, any>
  params?: Record<any, any>
  data?: any
  timeout?: number
}

export interface HttpResponse {
  status: number
  statusText: string
  headers?: Record<any, any>
  data?: any
}
