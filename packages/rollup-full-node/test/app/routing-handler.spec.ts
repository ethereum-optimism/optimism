/* External Imports */
import {
  add0x,
  JsonRpcError,
  JsonRpcErrorResponse,
  JsonRpcResponse,
  JsonRpcSuccessResponse,
  SimpleClient,
  sleep,
  TestUtils,
} from '@eth-optimism/core-utils'
/* Internal Imports */
import {
  AccountRateLimiter,
  allWeb3RpcMethodsIncludingTest,
  FormattedJsonRpcError,
  InvalidTransactionDesinationError,
  RateLimitError,
  TransactionLimitError,
  UnsupportedMethodError,
  Web3RpcMethods,
} from '../../src/types'
import { Wallet } from 'ethers'
import {
  getMethodsToRouteWithTransactionHandler,
  NoOpAccountRateLimiter,
  RoutingHandler,
} from '../../src/app'

class DummySimpleClient extends SimpleClient {
  constructor(private readonly cannedResponse: JsonRpcResponse) {
    super('')
  }

  public async makeRpcCall(
    method: string,
    params?: any
  ): Promise<JsonRpcResponse> {
    return this.cannedResponse
  }
}

class DummyRateLimiter implements AccountRateLimiter {
  constructor(
    public limitNextRequest: boolean = false,
    public limitNextTransaction: boolean = false
  ) {}

  public validateRateLimit(sourceIpAddress: string): void {
    if (this.limitNextRequest) {
      throw new RateLimitError(sourceIpAddress, 2, 1, 1000)
    }
  }

  public validateTransactionRateLimit(address: string): void {
    if (this.limitNextTransaction) {
      throw new TransactionLimitError(address, 2, 1, 1000)
    }
  }
}

const getSignedTransaction = async (
  to: string = Wallet.createRandom().address,
  wallet: Wallet = Wallet.createRandom()
) => {
  return wallet.sign({
    to,
    nonce: 0,
  })
}

const whitelistedIpAddress = '9.9.9.9'
const whitelistedIpAddress2 = '8.8.8.8'

const transactionResponse = 'transaction'
const readOnlyResponse = 'read only'
const transactionResponsePayload: JsonRpcSuccessResponse = {
  jsonrpc: '2.0',
  id: 123,
  result: transactionResponse,
}
const readOnlyPayload: JsonRpcSuccessResponse = {
  jsonrpc: '2.0',
  id: 1234,
  result: readOnlyResponse,
}

describe('Routing Handler', () => {
  describe('Routing Tests', () => {
    const routingHandler = new RoutingHandler(
      new DummySimpleClient(transactionResponsePayload),
      new DummySimpleClient(readOnlyPayload),
      '',
      new NoOpAccountRateLimiter(),
      [],
      [],
      new Set<string>(allWeb3RpcMethodsIncludingTest)
    )

    it('properly routes transactions vs other requests', async () => {
      const methods: string[] = Object.values(Web3RpcMethods)
      for (const method of methods) {
        const params: any[] = []
        if (method === Web3RpcMethods.sendRawTransaction) {
          params.push(await getSignedTransaction())
        }
        const res = await routingHandler.handleRequest(method, params, '')

        if (getMethodsToRouteWithTransactionHandler().indexOf(method) < 0) {
          res.should.equal(
            readOnlyResponse,
            `${method} should have been routed to read-only handler!`
          )
        } else {
          res.should.equal(
            transactionResponse,
            `${method} should have been routed to transaction handler!`
          )
        }
      }
    })
  })

  describe('Rate Limiter Tests', () => {
    let rateLimiter: DummyRateLimiter
    let routingHandler: RoutingHandler

    beforeEach(() => {
      rateLimiter = new DummyRateLimiter()
      routingHandler = new RoutingHandler(
        new DummySimpleClient(transactionResponsePayload),
        new DummySimpleClient(readOnlyPayload),
        '',
        rateLimiter,
        [whitelistedIpAddress],
        [],
        new Set<string>(allWeb3RpcMethodsIncludingTest)
      )
    })

    it('lets transactions through when not limited', async () => {
      // This should not throw
      await routingHandler.handleRequest(
        Web3RpcMethods.sendRawTransaction,
        [await getSignedTransaction()],
        ''
      )
    })

    it('lets requests through when not limited', async () => {
      // This should not throw
      await routingHandler.handleRequest(
        Web3RpcMethods.getBlockByNumber,
        ['0x0'],
        ''
      )
    })

    it('does not let transactions through when limited', async () => {
      rateLimiter.limitNextTransaction = true
      await TestUtils.assertThrowsAsync(async () => {
        return routingHandler.handleRequest(
          Web3RpcMethods.sendRawTransaction,
          [await getSignedTransaction()],
          ''
        )
      }, TransactionLimitError)
    })

    it('does not let requests through when limited', async () => {
      rateLimiter.limitNextRequest = true
      await TestUtils.assertThrowsAsync(async () => {
        return routingHandler.handleRequest(
          Web3RpcMethods.getBlockByNumber,
          ['0x0'],
          ''
        )
      }, RateLimitError)
    })

    it('lets transactions through when whitelisted', async () => {
      rateLimiter.limitNextTransaction = true
      await routingHandler.handleRequest(
        Web3RpcMethods.sendRawTransaction,
        [await getSignedTransaction()],
        whitelistedIpAddress
      )
    })

    it('lets requests through when whitelisted', async () => {
      rateLimiter.limitNextRequest = true
      await routingHandler.handleRequest(
        Web3RpcMethods.getBlockByNumber,
        ['0x0'],
        whitelistedIpAddress
      )
    })
  })

  describe('unsupported destination tests', () => {
    let routingHandler: RoutingHandler

    const deployerWallet: Wallet = Wallet.createRandom()
    const whitelistedTo: string = Wallet.createRandom().address

    beforeEach(() => {
      routingHandler = new RoutingHandler(
        new DummySimpleClient(transactionResponsePayload),
        new DummySimpleClient(readOnlyPayload),
        deployerWallet.address,
        new NoOpAccountRateLimiter(),
        [],
        [whitelistedTo],
        new Set<string>(allWeb3RpcMethodsIncludingTest)
      )
    })

    it('allows transactions to the whitelisted address', async () => {
      // Should not throw
      await routingHandler.handleRequest(
        Web3RpcMethods.sendRawTransaction,
        [await getSignedTransaction(whitelistedTo)],
        ''
      )
    })

    it('allows transactions to the whitelisted address from deployer address', async () => {
      // Should not throw
      await routingHandler.handleRequest(
        Web3RpcMethods.sendRawTransaction,
        [await getSignedTransaction(whitelistedTo, deployerWallet)],
        ''
      )
    })

    it('disallows transactions to non-whitelisted addresses', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        return routingHandler.handleRequest(
          Web3RpcMethods.sendRawTransaction,
          [await getSignedTransaction()],
          ''
        )
      }, InvalidTransactionDesinationError)
    })

    it('allows transactions to any address from deployer address', async () => {
      // Should not throw
      await routingHandler.handleRequest(
        Web3RpcMethods.sendRawTransaction,
        [
          await getSignedTransaction(
            Wallet.createRandom().address,
            deployerWallet
          ),
        ],
        ''
      )
    })
  })

  describe('unsupported methods tests', () => {
    let routingHandler: RoutingHandler

    beforeEach(() => {
      routingHandler = new RoutingHandler(
        new DummySimpleClient(transactionResponsePayload),
        new DummySimpleClient(readOnlyPayload),
        '',
        new NoOpAccountRateLimiter(),
        [],
        [],
        new Set<string>([Web3RpcMethods.sendRawTransaction])
      )
    })

    it('allows whitelisted method to be invoked', async () => {
      // Should not throw
      await routingHandler.handleRequest(
        Web3RpcMethods.sendRawTransaction,
        [await getSignedTransaction()],
        ''
      )
    })

    it('disallows whitelisted method to be invoked', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        return routingHandler.handleRequest(
          Web3RpcMethods.getBlockByNumber,
          ['0x0'],
          ''
        )
      }, UnsupportedMethodError)
    })
  })

  describe('Environment variable refresh', () => {
    let routingHandler: RoutingHandler
    let rateLimiter: DummyRateLimiter
    const whitelistedTo: string = add0x(Wallet.createRandom().address)

    beforeEach(() => {
      rateLimiter = new DummyRateLimiter()
      routingHandler = new RoutingHandler(
        new DummySimpleClient(transactionResponsePayload),
        new DummySimpleClient(readOnlyPayload),
        '',
        rateLimiter,
        ['0.0.0.0'],
        [whitelistedTo],
        new Set<string>([
          Web3RpcMethods.sendRawTransaction,
          Web3RpcMethods.networkVersion,
        ]),
        1_000
      )
    })

    describe('Contract deployer address', () => {
      let deployerWallet: Wallet
      beforeEach(() => {
        deployerWallet = Wallet.createRandom()
        process.env.CONTRACT_DEPLOYER_ADDRESS = add0x(deployerWallet.address)
      })
      afterEach(() => {
        delete process.env.CONTRACT_DEPLOYER_ADDRESS
      })

      it('allows transactions any address from deployer address', async () => {
        await sleep(1100)
        // Should not throw
        await routingHandler.handleRequest(
          Web3RpcMethods.sendRawTransaction,
          [
            await getSignedTransaction(
              Wallet.createRandom().address,
              deployerWallet
            ),
          ],
          ''
        )
      })

      it('disallows transactions to any address from non-deployer address', async () => {
        await sleep(1100)

        await TestUtils.assertThrowsAsync(async () => {
          return routingHandler.handleRequest(
            Web3RpcMethods.sendRawTransaction,
            [await getSignedTransaction()],
            ''
          )
        }, InvalidTransactionDesinationError)
      })
    })

    describe('To address whitelist', () => {
      let toAddress1: string
      let toAddress2: string
      beforeEach(() => {
        toAddress1 = add0x(Wallet.createRandom().address)
        toAddress2 = add0x(Wallet.createRandom().address)
        process.env.COMMA_SEPARATED_TO_ADDRESS_WHITELIST = [
          toAddress1,
          toAddress2,
        ].join(',')
      })
      afterEach(() => {
        delete process.env.COMMA_SEPARATED_TO_ADDRESS_WHITELIST
      })

      it('allows transactions to whitelisted addresses', async () => {
        await sleep(1100)
        // Should not throw
        await routingHandler.handleRequest(
          Web3RpcMethods.sendRawTransaction,
          [await getSignedTransaction(toAddress1)],
          ''
        )

        await routingHandler.handleRequest(
          Web3RpcMethods.sendRawTransaction,
          [await getSignedTransaction(toAddress2)],
          ''
        )
      })

      it('disallows transactions to non-whitelisted address', async () => {
        await sleep(1100)
        await TestUtils.assertThrowsAsync(async () => {
          await routingHandler.handleRequest(
            Web3RpcMethods.sendRawTransaction,
            [await getSignedTransaction(whitelistedTo)],
            ''
          )
        }, InvalidTransactionDesinationError)
      })
    })

    describe('Rate limit whitelisted IPs', () => {
      beforeEach(() => {
        process.env.COMMA_SEPARATED_RATE_LIMIT_WHITELISTED_IPS = [
          whitelistedIpAddress,
          whitelistedIpAddress2,
        ].join(',')
      })
      afterEach(() => {
        delete process.env.COMMA_SEPARATED_RATE_LIMIT_WHITELISTED_IPS
      })

      it('lets transactions through when not limited', async () => {
        await sleep(1100)
        // This should not throw
        await routingHandler.handleRequest(
          Web3RpcMethods.sendRawTransaction,
          [await getSignedTransaction(whitelistedTo)],
          ''
        )
      })

      it('does not let transactions through when limited', async () => {
        await sleep(1100)
        rateLimiter.limitNextTransaction = true
        await TestUtils.assertThrowsAsync(async () => {
          return routingHandler.handleRequest(
            Web3RpcMethods.sendRawTransaction,
            [await getSignedTransaction(whitelistedTo)],
            ''
          )
        }, TransactionLimitError)
      })

      it('lets transactions through when limited and whitelisted', async () => {
        await sleep(1100)
        rateLimiter.limitNextTransaction = true
        // This should not throw
        await routingHandler.handleRequest(
          Web3RpcMethods.sendRawTransaction,
          [await getSignedTransaction(whitelistedTo)],
          whitelistedIpAddress
        )
      })

      it('lets requests through when not limited', async () => {
        await sleep(1100)
        // This should not throw
        await routingHandler.handleRequest(
          Web3RpcMethods.networkVersion,
          [],
          ''
        )
      })

      it('does not let requests through when limited', async () => {
        await sleep(1100)
        rateLimiter.limitNextRequest = true
        await TestUtils.assertThrowsAsync(async () => {
          return routingHandler.handleRequest(
            Web3RpcMethods.networkVersion,
            [],
            ''
          )
        }, RateLimitError)
      })

      it('lets requests through when limited and whitelisted', async () => {
        await sleep(1100)
        rateLimiter.limitNextRequest = true
        // This should not throw
        await routingHandler.handleRequest(
          Web3RpcMethods.networkVersion,
          [],
          whitelistedIpAddress
        )
      })
    })
  })

  describe('Formatted JSON RPC Responses', () => {
    let routingHandler: RoutingHandler

    const txError: JsonRpcError = {
      code: -123,
      message: 'tx error',
      data: 'tx error',
    }

    const roError: JsonRpcError = {
      code: -1234,
      message: 'r/o error',
      data: 'r/o error',
    }

    const transactionErrorResponsePayload: JsonRpcErrorResponse = {
      jsonrpc: '2.0',
      id: 123,
      error: txError,
    }
    const readOnlyErrorResponsePayload: JsonRpcErrorResponse = {
      jsonrpc: '2.0',
      id: 1234,
      error: roError,
    }

    beforeEach(() => {
      routingHandler = new RoutingHandler(
        new DummySimpleClient(transactionErrorResponsePayload),
        new DummySimpleClient(readOnlyErrorResponsePayload),
        '',
        new NoOpAccountRateLimiter()
      )
    })

    it('throws Json error on transaction', async () => {
      const error: Error = await TestUtils.assertThrowsAsync(async () => {
        await routingHandler.handleRequest(
          Web3RpcMethods.sendRawTransaction,
          [await getSignedTransaction()],
          ''
        )
      })

      error.should.be.instanceOf(FormattedJsonRpcError, 'Invalid error type!')
      const formatted: FormattedJsonRpcError = error as FormattedJsonRpcError
      formatted.jsonRpcResponse.should.deep.equal(
        transactionErrorResponsePayload,
        'Incorrect error returned!'
      )
    })

    it('throws Json error on read only request', async () => {
      const error: Error = await TestUtils.assertThrowsAsync(async () => {
        await routingHandler.handleRequest(
          Web3RpcMethods.networkVersion,
          [],
          ''
        )
      })

      error.should.be.instanceOf(FormattedJsonRpcError, 'Invalid error type!')
      const formatted: FormattedJsonRpcError = error as FormattedJsonRpcError
      formatted.jsonRpcResponse.should.deep.equal(
        readOnlyErrorResponsePayload,
        'Incorrect error returned!'
      )
    })
  })
})
