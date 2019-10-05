import { SimpleServer } from '@pigi/core'
import { RollupAggregator } from './rollup-aggregator'
import {
  Address,
  AGGREGATOR_API,
  SignedStateReceipt,
  SignedTransaction,
  UNISWAP_ADDRESS,
} from '../index'

export class AggregatorServer extends SimpleServer {
  public constructor(
    aggregator: RollupAggregator,
    hostname: string,
    port: number,
    middleware?: Function[]
  ) {
    // REST API for our aggregator
    const methods = {
      [AGGREGATOR_API.getState]: async (
        account: Address
      ): Promise<SignedStateReceipt> => aggregator.getState(account),

      [AGGREGATOR_API.getUniswapState]: async (): Promise<SignedStateReceipt> =>
        aggregator.getState(UNISWAP_ADDRESS),

      [AGGREGATOR_API.applyTransaction]: async (
        signedTransaction: SignedTransaction
      ): Promise<SignedStateReceipt[]> =>
        aggregator.applyTransaction(signedTransaction),

      [AGGREGATOR_API.requestFaucetFunds]: async (
        signedTransaction: SignedTransaction
      ): Promise<SignedStateReceipt> =>
        aggregator.requestFaucetFunds(signedTransaction),

      [AGGREGATOR_API.getTransactionCount]: async (): Promise<number> =>
        aggregator.getTransactionCount(),
    }
    super(methods, hostname, port, middleware)
  }
}
