/* External Imports */
import {
  AxiosHttpClient,
  getLogger,
  JSONRPC_ERRORS,
} from '@eth-optimism/core-utils/build/src'
import { AxiosResponse } from 'axios'

/* Internal Imports */
import { FullnodeRpcServer } from '../../src/app'
import {
  FullnodeHandler,
  RevertError,
  UnsupportedMethodError,
} from '../../src/types'
import { should } from '../setup'

const log = getLogger('fullnode-rpc-server', true)

const dummyResponse: string = 'Dummy Response =D'

const unsupportedMethod: string = 'unsupported!'
const revertMethod: string = 'revert!'
class DummyFullnodeHandler implements FullnodeHandler {
  public async handleRequest(
    method: string,
    params: string[]
  ): Promise<string> {
    if (method === unsupportedMethod) {
      throw new UnsupportedMethodError()
    }
    if (method === revertMethod) {
      throw new RevertError()
    }
    return dummyResponse
  }
}

const defaultSupportedMethods: Set<string> = new Set([
  'valid',
  'should',
  'work',
])

const request = async (
  client: AxiosHttpClient,
  payload: {}
): Promise<AxiosResponse> => {
  return client.request({
    url: '',
    method: 'post',
    data: payload,
  })
}

const getBadPayloads = () => {
  return [
    {
      jsonrpc: '2.0',
      method: defaultSupportedMethods[0],
    },
    {
      id: '1',
      jsonrpc: '2.0',
      method: defaultSupportedMethods[0],
    },
    {
      id: 1,
      method: defaultSupportedMethods[0],
    },
    {
      id: 1,
      jsonrpc: 2.0,
      method: defaultSupportedMethods[0],
    },
    { id: 1, jsonrpc: '2.0' },
    { id: 1, jsonrpc: '2.0', method: unsupportedMethod },
  ]
}

const host = '0.0.0.0'
const port = 9999

describe('FullnodeHandler RPC Server', () => {
  const fullnodeHandler: FullnodeHandler = new DummyFullnodeHandler()
  let fullnodeRpcServer: FullnodeRpcServer
  let baseUrl: string
  let client: AxiosHttpClient

  beforeEach(() => {
    fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

    fullnodeRpcServer.listen()

    baseUrl = `http://${host}:${port}`
    client = new AxiosHttpClient(baseUrl)
  })

  afterEach(() => {
    if (!!fullnodeRpcServer) {
      fullnodeRpcServer.close()
    }
  })

  describe('single requests', () => {
    it('should work for valid requests & methods', async () => {
      const results: AxiosResponse[] = await Promise.all(
        Array.from(defaultSupportedMethods).map((x) =>
          request(client, { id: 1, jsonrpc: '2.0', method: x })
        )
      )

      results.forEach((r) => {
        r.status.should.equal(200)

        r.data.should.haveOwnProperty('id')
        r.data['id'].should.equal(1)

        r.data.should.haveOwnProperty('jsonrpc')
        r.data['jsonrpc'].should.equal('2.0')

        r.data.should.haveOwnProperty('result')
        r.data['result'].should.equal(dummyResponse)
      })
    })

    it('fails on bad format or method', async () => {
      const results: AxiosResponse[] = await Promise.all(
        getBadPayloads().map((x) => request(client, x))
      )
      results.forEach((r) => {
        r.status.should.equal(200)

        r.data.should.haveOwnProperty('id')
        r.data.should.haveOwnProperty('jsonrpc')
        r.data.should.haveOwnProperty('error')

        r.data['jsonrpc'].should.equal('2.0')
      })
    })

    it('reverts properly', async () => {
      const result: AxiosResponse = await request(client, {
        id: 1,
        jsonrpc: '2.0',
        method: revertMethod,
      })

      result.status.should.equal(200)

      result.data.should.haveOwnProperty('id')
      result.data.should.haveOwnProperty('jsonrpc')
      result.data.should.haveOwnProperty('error')
      result.data['error'].should.haveOwnProperty('message')
      result.data['error'].should.haveOwnProperty('code')
      result.data['error']['message'].should.equal(
        JSONRPC_ERRORS.REVERT_ERROR.message
      )
      result.data['error']['code'].should.equal(
        JSONRPC_ERRORS.REVERT_ERROR.code
      )

      result.data['jsonrpc'].should.equal('2.0')
    })
  })

  describe('batch requests', () => {
    it('should work for valid requests & methods', async () => {
      const batchRequest = Array.from(
        defaultSupportedMethods
      ).map((method, id) => ({ jsonrpc: '2.0', id, method }))
      const result: AxiosResponse = await request(client, batchRequest)

      result.status.should.equal(200)
      const results = result.data

      results.forEach((r, id) => {
        r.should.haveOwnProperty('id')
        r['id'].should.equal(id)

        r.should.haveOwnProperty('jsonrpc')
        r['jsonrpc'].should.equal('2.0')

        r.should.haveOwnProperty('result')
        r['result'].should.equal(dummyResponse)
      })
    })
    it('should fail on bad format or for valid requests & methods', async () => {
      const result: AxiosResponse = await request(client, getBadPayloads())

      result.status.should.equal(200)
      const results = result.data

      results.forEach((r) => {
        r.should.haveOwnProperty('id')
        r.should.haveOwnProperty('jsonrpc')
        r.should.haveOwnProperty('error')

        r['jsonrpc'].should.equal('2.0')
      })
    })
    it('should not allow batches of batches', async () => {
      const batchOfBatches = Array.from(
        defaultSupportedMethods
      ).map((method, id) => [{ jsonrpc: '2.0', id, method }])
      const result: AxiosResponse = await request(client, batchOfBatches)

      result.status.should.equal(200)
      const results = result.data

      results.forEach((r) => {
        r.should.haveOwnProperty('id')
        r.should.haveOwnProperty('jsonrpc')
        r.should.haveOwnProperty('error')
        r['error'].should.haveOwnProperty('message')
        r['error'].should.haveOwnProperty('code')
        r['error']['message'].should.equal(
          JSONRPC_ERRORS.INVALID_REQUEST.message
        )
        r['error']['code'].should.equal(JSONRPC_ERRORS.INVALID_REQUEST.code)

        r['jsonrpc'].should.equal('2.0')
      })
    })
  })
})
