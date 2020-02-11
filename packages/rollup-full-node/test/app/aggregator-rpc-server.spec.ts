/* External Imports */
import {
  AxiosHttpClient,
  JsonRpcClient,
  JsonRpcRequest,
  JsonRpcResponse,
  JsonRpcSuccessResponse,
  RpcClient,
  SimpleClient,
} from '@eth-optimism/core-utils/build/src'
import { AxiosResponse } from 'axios'

/* Internal Imports */
import { AggregatorRpcServer } from '../../src/app/aggregator-rpc-server'
import { Aggregator } from '../../src/types'
import { should } from '../setup'

const dummyResponse: string = 'Dummy Response =D'

class DummyAggregator implements Aggregator {
  public async handleRequest(req: JsonRpcRequest): Promise<JsonRpcResponse> {
    return {
      id: req.id,
      jsonrpc: req.jsonrpc,
      result: dummyResponse,
    }
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

describe('Aggregator RPC Server', () => {
  const aggregator: Aggregator = new DummyAggregator()
  let aggregatorServer: AggregatorRpcServer
  let baseUrl: string
  let client: AxiosHttpClient

  beforeEach(() => {
    aggregatorServer = new AggregatorRpcServer(
      defaultSupportedMethods,
      aggregator,
      host,
      port
    )

    aggregatorServer.listen()

    baseUrl = `http://${host}:${port}`
    client = new AxiosHttpClient(baseUrl)
  })

  afterEach(() => {
    if (!!aggregatorServer) {
      aggregatorServer.close()
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
      request(client, { id: 1, jsonrpc: '2.0', method: 'notValid' }),
    ])

    results.forEach((r) => {
      r.status.should.equal(200)

      r.data.should.haveOwnProperty('id')
      r.data.should.haveOwnProperty('jsonrpc')
      r.data.should.haveOwnProperty('error')

      r.data['jsonrpc'].should.equal('2.0')
    })
  })
})
