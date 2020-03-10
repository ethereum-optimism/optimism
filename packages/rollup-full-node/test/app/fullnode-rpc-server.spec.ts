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
    const results: AxiosResponse[] = await Promise.all([
      request(client, { jsonrpc: '2.0', method: defaultSupportedMethods[0] }),
      request(client, {
        id: '1',
        jsonrpc: '2.0',
        method: defaultSupportedMethods[0],
      }),
      request(client, { id: 1, method: defaultSupportedMethods[0] }),
      request(client, {
        id: 1,
        jsonrpc: 2.0,
        method: defaultSupportedMethods[0],
      }),
      request(client, { id: 1, jsonrpc: '2.0' }),
      request(client, { id: 1, jsonrpc: '2.0', method: unsupportedMethod }),
    ])

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
    result.data['error']['code'].should.equal(JSONRPC_ERRORS.REVERT_ERROR.code)

    result.data['jsonrpc'].should.equal('2.0')
  })
})
