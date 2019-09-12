/* External Imports */
import {
  SignatureVerifier,
  DefaultSignatureVerifier,
  SimpleServer,
  serializeObject,
  DefaultSignatureProvider,
  DB,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  SignedTransaction,
  State,
  Balances,
  TransactionReceipt,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  generateTransferTx,
  AGGREGATOR_API,
  Transaction,
  SignatureProvider,
  UNISWAP_ADDRESS,
  AGGREGATOR_ADDRESS,
} from '../index'
import { ethers } from 'ethers'
import { RollupStateMachine } from '../types'
import { DefaultRollupStateMachine } from '../rollup-state-machine'

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
  private readonly rollupStateMachine: RollupStateMachine

  constructor(
    rollupStateMachine: RollupStateMachine,
    hostname: string,
    port: number,
    mnemonic: string,
    signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance(),
    middleware?: Function[]
  ) {
    const wallet: ethers.Wallet = ethers.Wallet.fromMnemonic(mnemonic)
    const signatureProvider: SignatureProvider = new DefaultSignatureProvider(
      wallet
    )

    // REST API for our aggregator
    const methods = {
      /*
       * Get balances for some account
       */
      [AGGREGATOR_API.getBalances]: async (
        account: Address
      ): Promise<Balances> => rollupStateMachine.getBalances(account),

      /*
       * Get balances for Uniswap
       */
      [AGGREGATOR_API.getUniswapBalances]: async (): Promise<Balances> =>
        rollupStateMachine.getBalances(UNISWAP_ADDRESS),

      /*
       * Apply either a transfer or swap transaction
       */
      [AGGREGATOR_API.applyTransaction]: async (
        transaction: SignedTransaction
      ): Promise<TransactionReceipt> => {
        const stateUpdate: State = await rollupStateMachine.applyTransaction(
          transaction
        )
        const aggregatorSignature: string = await signatureProvider.sign(
          AGGREGATOR_ADDRESS,
          serializeObject(stateUpdate)
        )
        return {
          aggregatorSignature,
          stateUpdate,
        }
      },

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
        for (const tx of faucetTxs) {
          await rollupStateMachine.applyTransaction(tx)
        }

        // Return our new account balance
        return rollupStateMachine.getBalances(recipient)
      },
    }
    super(methods, hostname, port, middleware)
    this.rollupStateMachine = rollupStateMachine
  }
}
