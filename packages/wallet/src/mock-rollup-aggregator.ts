/* External Imports */
import { SimpleServer } from '@pigi/core'

/* Internal Imports */
import {
  Address,
  SignedTransaction,
  State,
  MockRollupStateMachine,
  Balances,
  TransactionReceipt,
  UNISWAP_ADDRESS,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  AGGREGATOR_ADDRESS,
  TokenType,
  generateTransferTx,
  AGGREGATOR_API,
} from '.'

/*
 * Generate two transactions which together send the user some UNI
 * & some PIGI
 */
const generateFaucetTxs = (
  recipient: Address,
  amount: number
): [SignedTransaction, SignedTransaction] => {
  return [
    {
      signature: AGGREGATOR_ADDRESS,
      transaction: generateTransferTx(recipient, UNI_TOKEN_TYPE, amount),
    },
    {
      signature: AGGREGATOR_ADDRESS,
      transaction: generateTransferTx(recipient, PIGI_TOKEN_TYPE, amount),
    },
  ]
}

/*
 * A mock aggregator implementation which allows for transfers, swaps,
 * balance queries, & faucet requests
 */
export class MockAggregator extends SimpleServer {
  public rollupStateMachine: MockRollupStateMachine

  constructor(
    genesisState: State,
    hostname: string,
    port: number,
    middleware?: Function[]
  ) {
    const rollupStateMachine = new MockRollupStateMachine(genesisState)

    // REST API for our aggregator
    const methods = {
      /*
       * Get balances for some account
       */
      [AGGREGATOR_API.getBalances]: (account: Address): Balances =>
        rollupStateMachine.getBalances(account),

      /*
       * Get balances for Uniswap
       */
      [AGGREGATOR_API.getUniswapBalances]: (): Balances =>
        rollupStateMachine.getUniswapBalances(),

      /*
       * Apply either a transfer or swap transaction
       */
      [AGGREGATOR_API.applyTransaction]: (
        transaction: SignedTransaction
      ): TransactionReceipt => rollupStateMachine.applyTransaction(transaction),

      /*
       * Request money from a faucet
       */
      [AGGREGATOR_API.requestFaucetFunds]: (
        params: [Address, number]
      ): Balances => {
        const [recipient, amount] = params
        // Generate the faucet txs (one sending uni the other pigi)
        const faucetTxs = generateFaucetTxs(recipient, amount)
        // Apply the two txs
        rollupStateMachine.applyTransaction(faucetTxs[0])
        rollupStateMachine.applyTransaction(faucetTxs[1])
        // Return our new account balance
        return rollupStateMachine.getBalances(recipient)
      },
    }
    super(methods, hostname, port, middleware)
    this.rollupStateMachine = rollupStateMachine
  }
}
