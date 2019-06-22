import '../../setup'

/* External Imports */
import debug from 'debug'
const log = debug('test:info:state-manager')

/* Internal Imports */
import { JsonRpcClient, JsonRpcHttpAdapter, AxiosHttpClient,  JsonRpcServer } from 'src/app'
import { HttpRequest, HttpResponse } from 'src/interfaces'

describe.only('JSON RPC Server', () => {
  describe('Server', () => {
    it('should initialize without throwing', async () => {
      const testFn = () => {
        log('We called it!')
      }
      const server = new JsonRpcServer({
        test: testFn
      }, 'localhost', 3000)
      await server.listen()
      const client = new JsonRpcClient<HttpRequest, HttpResponse>(new JsonRpcHttpAdapter(), new AxiosHttpClient('http://127.0.0.1:3000'))
      await client.handle('test', 'test123')
    })
  })
})
