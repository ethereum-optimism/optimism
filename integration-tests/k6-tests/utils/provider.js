import http from 'k6/http'

export class K6RpcProvider {
  constructor(url) {
    this.url = url
    this._nextId = 0
  }

  send(method, params = []) {
    const response = http.post(
      this.url,
      JSON.stringify({
        method: method,
        params: params,
        id: this._nextId++,
        jsonrpc: '2.0',
      }),
      {
        headers: {
          'Content-Type': 'application/json'
        },
      }
    )

    return response.json()
  }
}
