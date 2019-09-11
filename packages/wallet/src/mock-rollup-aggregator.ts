/* External Imports */
import {
  SignatureVerifier,
  DefaultSignatureVerifier,
  SimpleServer,
  serializeObject,
  DefaultSignatureProvider,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  SignedTransaction,
  State,
  MockRollupStateMachine,
  Balances,
  TransactionReceipt,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  AGGREGATOR_ADDRESS,
  generateTransferTx,
  AGGREGATOR_API,
  Transaction,
  SignatureProvider,
} from '.'
import { ethers } from 'ethers'

/*
 * Generate two transactions which together send the user some UNI
 * & some PIGI
 */
const generateFaucetTxs = async (
  recipient: Address,
  amount: number,
  aggregatorAddress: string = AGGREGATOR_ADDRESS,
  signatureProvider?: SignatureProvider
): Promise<SignedTransaction[]> => {
  const txOne: Transaction = generateTransferTx(
    recipient,
    UNI_TOKEN_TYPE,
    amount
  )
  const txTwo: Transaction = generateTransferTx(
    recipient,
    PIGI_TOKEN_TYPE,
    amount
  )

  return [
    {
      signature: await signatureProvider.sign(
        aggregatorAddress,
        serializeObject(txOne)
      ),
      transaction: txOne,
    },
    {
      signature: await signatureProvider.sign(
        aggregatorAddress,
        serializeObject(txTwo)
      ),
      transaction: txTwo,
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
    mnemonic: string,
    signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance(),
    middleware?: Function[]
  ) {
    const rollupStateMachine = new MockRollupStateMachine(
      genesisState,
      signatureVerifier
    )

    const wallet: ethers.Wallet = ethers.Wallet.fromMnemonic(mnemonic)
    const signatureProvider: SignatureProvider = new DefaultSignatureProvider(
      wallet
    )

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
      [AGGREGATOR_API.requestFaucetFunds]: async (
        params: [Address, number]
      ): Promise<Balances> => {
        const [recipient, amount] = params
        // Generate the faucet txs (one sending uni the other pigi)
        const faucetTxs = await generateFaucetTxs(
          recipient,
          amount,
          wallet.address,
          signatureProvider
        )
        // Apply the two txs
        faucetTxs.forEach((tx) => rollupStateMachine.applyTransaction(tx))

        // Return our new account balance
        return rollupStateMachine.getBalances(recipient)
      },
    }
    super(methods, hostname, port, middleware)
    this.rollupStateMachine = rollupStateMachine
  }
}
