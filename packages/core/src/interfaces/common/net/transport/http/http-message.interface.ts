export interface HttpRequest {
  url: string
  method:
    | 'get'
    | 'GET'
    | 'head'
    | 'HEAD'
    | 'post'
    | 'POST'
    | 'put'
    | 'PUT'
    | 'delete'
    | 'DELETE'
    | 'options'
    | 'OPTIONS'
    | 'patch'
    | 'PATCH'
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
