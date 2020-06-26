import {
  add0x,
  buildJsonRpcError,
  ExpressHttpServer,
  getLogger,
  isObject,
  JsonRpcRequest,
  JsonRpcResponse,
  logError,
  Logger,
} from '@eth-optimism/core-utils'
import { JsonRpcProvider } from 'ethers/providers'
import { keccak256 } from 'ethers/utils'
import { monkeyPatchL2Provider } from '../../src/app'

const nodeLogger: Logger = getLogger('mock-node', true)

const port = 9991
const hostname = '0.0.0.0'
const senderAddress = '0xa7d9ddbe1f17865597fbd27ec712455208b6b76d'

class MockNode extends ExpressHttpServer {
  constructor(_hostname: string = hostname, _port: number = port) {
    super(_port, _hostname)
  }

  protected initRoutes(): void {
    this.app.post('/', async (req, res) => {
      return res.json(await this.handleRequest(req))
    })
  }

  /**
   * Handles the provided request, returning the appropriate response object
   * @param req The request to handle
   * @returns The JSON-stringifiable response object.
   */
  protected async handleRequest(
    req: any
  ): Promise<JsonRpcResponse | JsonRpcResponse[]> {
    let request: JsonRpcRequest

    try {
      request = req.body

      const txResponse = {
        blockHash:
          '0x1d59ff54b1eb26b013ce3cb5fc9dab3705b415a67127a003c3e61eb445bb8df2',
        blockNumber: '0x5daf3b', // 6139707
        from: '0xa7d9ddbe1f17865597fbd27ec712455208b6b76d',
        gas: '0xc350', // 50000
        gasPrice: '0x4a817c800', // 20000000000
        hash:
          '0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b',
        input: '0x68656c6c6f21',
        nonce: '0x15', // 21
        to: '0xf02c1c8e6114b1dbe8937a39260b5b0a374432bb',
        transactionIndex: '0x41', // 65
        value: '0xf3dbb76162000', // 4290000000000000
        v: '0x25', // 37
        r: '0x1b5e176d927f8e9ab405058b2d2457392da3e20f328b16ddabcebc33eaac5fea',
        s: '0x4ba69724e8f69de52f0125ad8b3c5c2cef33019bac3249e2c0a2192766d1721c',
        l1MessageSender: senderAddress,
      }

      let result
      switch (request.method) {
        case 'net_version':
          result = '108'
          break
        case 'eth_blockNumber':
          result = '0x0'
          break
        case 'eth_getTransactionByHash':
          result = txResponse
          break
        case 'eth_getBlockByHash':
          result = {
            difficulty: '0x4ea3f27bc',
            extraData:
              '0x476574682f4c5649562f76312e302e302f6c696e75782f676f312e342e32',
            gasLimit: '0x1388',
            gasUsed: '0x0',
            hash:
              '0xdc0818cf78f21a8e70579cb46a43643f78291264dda342ae31049421c82d21ae',
            logsBloom:
              '0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
            miner: '0xbb7b8287f3f0a933474a79eae42cbca977791171',
            mixHash:
              '0x4fffe9ae21f1c9e15207b1f472d5bbdd68c9595d461666602f2be20daf5e7843',
            nonce: '0x689056015818adbe',
            number: '0x1b4',
            parentHash:
              '0xe99e022112df268087ea7eafaf4790497fd21dbeeb6bd7a1721df161a6657a54',
            receiptsRoot:
              '0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421',
            sha3Uncles:
              '0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347',
            size: '0x220',
            stateRoot:
              '0xddc8b0234c2e0cad087c8b389aa7ef01f7d79b2570bccb77ce48648aa61c904d',
            timestamp: '0x55ba467c',
            totalDifficulty: '0x78ed983323d',
            transactions: [txResponse],
            transactionsRoot:
              '0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421',
            uncles: [],
          }
          break
        default:
          throw Error(`got unsupported method ${request.method}`)
      }
      return {
        id: request.id,
        jsonrpc: request.jsonrpc,
        result,
      }
    } catch (err) {
      logError(nodeLogger, `Uncaught exception at endpoint-level`, err)
      return buildJsonRpcError(
        'INTERNAL_ERROR',
        request && request.id ? request.id : null
      )
    }
  }
}

describe('L1MessageSender Tests', () => {
  let server: MockNode
  let provider: JsonRpcProvider
  before(() => {
    server = new MockNode()
    server.listen()
  })
  after(async () => {
    await server.close()
  })

  beforeEach(() => {
    provider = new JsonRpcProvider(`http://${hostname}:${port}`)
  })

  it('should fail to add l1MessageSender to Transactions', async () => {
    const tx = await provider.getTransaction(keccak256('0xdeadb33f'))

    let truth = !!tx
    truth.should.equal(true, 'Tx response should exist!')
    truth = !tx['l1MessageSender']
    truth.should.equal(true, 'L1 Message Sender exists when it should not!')
  })

  it('should fail to add l1MessageSender to block Transactions', async () => {
    const block = await provider.getBlock(keccak256('0xdeadb33f'), true)

    let truth = !!block
    truth.should.equal(true, 'Block response should exist!')
    truth =
      block.transactions !== undefined &&
      block.transactions.length > 0 &&
      isObject(block.transactions[0])
    truth.should.equal(true, 'Block should have a transaction!')
    truth = !block.transactions[0]['l1MessageSender']
    truth.should.equal(true, 'Block tx L1 Message Sender should not exist!')
  })

  describe('after fixing tx parsing', () => {
    beforeEach(() => {
      provider = monkeyPatchL2Provider(provider)
    })

    it('should fail to add l1MessageSender to Transactions', async () => {
      const tx = await provider.getTransaction(keccak256('0xdeadb33f'))

      let truth = !!tx
      truth.should.equal(true, 'Tx response should exist!')
      truth = !!tx['l1MessageSender']
      truth.should.equal(true, 'L1 Message Sender should exist!')
      tx['l1MessageSender'].should.equal(
        senderAddress,
        'L1 Message Sender address does not match!'
      )
    })

    it('should fail to add l1MessageSender to block Transactions', async () => {
      const block = await provider.getBlock(keccak256('0xdeadb33f'), true)

      let truth = !!block
      truth.should.equal(true, 'Block response should exist!')
      truth =
        block.transactions !== undefined &&
        block.transactions.length > 0 &&
        isObject(block.transactions[0])
      truth.should.equal(true, 'Block should have a transaction!')
      truth = !!block.transactions[0]['l1MessageSender']
      truth.should.equal(true, 'Block tx L1 Message Sender should exist!')
      block.transactions[0]['l1MessageSender'].should.equal(
        senderAddress,
        'L1 Message Sender address does not match!'
      )
    })
  })
})
