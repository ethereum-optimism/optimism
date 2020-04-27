/* External Imports */
import {
  bufToHexString,
  SimpleClient,
  TestUtils,
} from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  getMethodsToRouteWithTransactionHandler,
  RoutingHandler,
} from '../../src/app/handler'
import { NoOpAccountRateLimiter } from '../../src/app/utils'
import {
  AccountRateLimiter,
  allWeb3RpcMethodsIncludingTest,
  InvalidTransactionDesinationError,
  RateLimitError,
  TransactionLimitError,
  UnsupportedMethodError,
  Web3RpcMethods,
} from '../../src/types'
import { Wallet } from 'ethers'

class DummySimpleClient extends SimpleClient {
  constructor(private readonly cannedResponse: any) {
    super('')
  }
  public async handle<T>(method: string, params?: any): Promise<T> {
    return this.cannedResponse as T
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

describe.only('Routing Handler', () => {
  describe('Routing Tests', () => {
    const transactionResponse = 'transaction'
    const readOnlyResponse = 'read only'
    const routingHandler = new RoutingHandler(
      new DummySimpleClient(transactionResponse),
      new DummySimpleClient(readOnlyResponse),
      '',
      new NoOpAccountRateLimiter(),
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

    const transactionResponse = 'transaction'
    const readOnlyResponse = 'read only'

    beforeEach(() => {
      rateLimiter = new DummyRateLimiter()
      routingHandler = new RoutingHandler(
        new DummySimpleClient(transactionResponse),
        new DummySimpleClient(readOnlyResponse),
        '',
        rateLimiter,
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

    it('does not let transactions through when not limited', async () => {
      rateLimiter.limitNextTransaction = true
      await TestUtils.assertThrowsAsync(async () => {
        return routingHandler.handleRequest(
          Web3RpcMethods.sendRawTransaction,
          [await getSignedTransaction()],
          ''
        )
      }, TransactionLimitError)
    })

    it('does not let requests through when not limited', async () => {
      rateLimiter.limitNextRequest = true
      await TestUtils.assertThrowsAsync(async () => {
        return routingHandler.handleRequest(
          Web3RpcMethods.getBlockByNumber,
          ['0x0'],
          ''
        )
      }, RateLimitError)
    })
  })

  describe('unsupported destination tests', () => {
    let routingHandler: RoutingHandler

    const transactionResponse = 'transaction'
    const readOnlyResponse = 'read only'
    const deployerWallet: Wallet = Wallet.createRandom()
    const whitelistedTo: string = Wallet.createRandom().address

    beforeEach(() => {
      routingHandler = new RoutingHandler(
        new DummySimpleClient(transactionResponse),
        new DummySimpleClient(readOnlyResponse),
        deployerWallet.address,
        new NoOpAccountRateLimiter(),
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

    it('allows transactions to the whitelisted address from deployer address', async () => {
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

    const transactionResponse = 'transaction'
    const readOnlyResponse = 'read only'

    beforeEach(() => {
      routingHandler = new RoutingHandler(
        new DummySimpleClient(transactionResponse),
        new DummySimpleClient(readOnlyResponse),
        '',
        new NoOpAccountRateLimiter(),
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
})
