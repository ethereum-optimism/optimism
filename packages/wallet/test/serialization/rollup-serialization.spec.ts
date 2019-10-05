import '../setup'

/* External Imports */
import {
  SignedTransaction,
  Swap,
  RollupTransaction,
  Transfer,
  State,
  StateReceipt,
  SwapTransition,
  InclusionProof,
  TransferTransition,
  CreateAndTransferTransition,
  abiEncodeSignedTransaction,
  abiEncodeState,
  abiEncodeStateReceipt,
  abiEncodeTransaction,
  abiEncodeTransition,
  parseSignedTransactionFromABI,
  parseStateFromABI,
  parseStateReceiptFromABI,
  parseTransactionFromABI,
  parseTransitionFromABI,
  AGGREGATOR_ADDRESS,
  PIGI_TOKEN_TYPE,
  UNI_TOKEN_TYPE,
} from '../../src/'
import { BOB_ADDRESS } from '../helpers'

/* Internal Imports */

const stateRoot: string =
  '9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658'
const inclusionProof: InclusionProof = [stateRoot, stateRoot, stateRoot]

describe('RollupEncoding', () => {
  describe('Transactions', () => {
    it('should encoded & decode Transfer without throwing', async () => {
      const address = '0x' + '31'.repeat(20)
      const tx: Transfer = {
        sender: address,
        recipient: address,
        tokenType: 1,
        amount: 15,
      }

      const abiEncoded: string = abiEncodeTransaction(tx)
      const transfer: RollupTransaction = parseTransactionFromABI(abiEncoded)

      transfer.should.deep.equal(tx)
    })

    it('should encoded & decode Swap without throwing', async () => {
      const address = '0x' + '31'.repeat(20)
      const tx: Swap = {
        sender: address,
        tokenType: 1,
        inputAmount: 15,
        minOutputAmount: 4,
        timeout: +new Date(),
      }

      const abiEncoded: string = abiEncodeTransaction(tx)
      const swap: RollupTransaction = parseTransactionFromABI(abiEncoded)

      swap.should.deep.equal(tx)
    })

    it('should encoded & decode SignedTransactions without throwing', async () => {
      const address = '0x' + '31'.repeat(20)
      const transfer: Transfer = {
        sender: address,
        recipient: address,
        tokenType: 1,
        amount: 15,
      }
      const signedTransfer: SignedTransaction = {
        signature: '0x1234',
        transaction: transfer,
      }

      const swap: Swap = {
        sender: address,
        tokenType: 1,
        inputAmount: 15,
        minOutputAmount: 4,
        timeout: +new Date(),
      }
      const signedSwap: SignedTransaction = {
        signature: '0x4321',
        transaction: swap,
      }

      const abiEncodedSwap: string = abiEncodeSignedTransaction(signedSwap)
      const abiEncodedTransfer: string = abiEncodeSignedTransaction(
        signedTransfer
      )
      abiEncodedSwap.should.not.equal(abiEncodedTransfer)

      const parsedSwap: SignedTransaction = parseSignedTransactionFromABI(
        abiEncodedSwap
      )
      const parsedTransfer: SignedTransaction = parseSignedTransactionFromABI(
        abiEncodedTransfer
      )
      parsedSwap.should.not.deep.equal(parsedTransfer)

      parsedSwap.should.deep.equal(signedSwap)
      parsedTransfer.should.deep.equal(signedTransfer)
    })
  })

  describe('State', () => {
    it('should encoded & decode State without throwing', async () => {
      const state: State = {
        pubkey: BOB_ADDRESS,
        balances: {
          [UNI_TOKEN_TYPE]: 50,
          [PIGI_TOKEN_TYPE]: 100,
        },
      }

      const stateString: string = abiEncodeState(state)
      const parsedState: State = parseStateFromABI(stateString)

      parsedState.should.deep.equal(state)
    })
  })

  describe('State Receipt', () => {
    it('should encoded & decode StateReceipt without throwing', async () => {
      const state: State = {
        pubkey: BOB_ADDRESS,
        balances: {
          [UNI_TOKEN_TYPE]: 50,
          [PIGI_TOKEN_TYPE]: 100,
        },
      }
      const stateReceipt: StateReceipt = {
        slotIndex: 0,
        stateRoot,
        inclusionProof,
        blockNumber: 1,
        transitionIndex: 2,
        state,
      }

      const stateReceiptString: string = abiEncodeStateReceipt(stateReceipt)
      const parsedStateReceipt: StateReceipt = parseStateReceiptFromABI(
        stateReceiptString
      )

      parsedStateReceipt.should.deep.equal(stateReceipt)
    })
  })

  describe('Transitions', () => {
    const sig = '0x1234'

    it('should encoded & decode Swap Transition without throwing', async () => {
      const transition: SwapTransition = {
        stateRoot,
        senderSlotIndex: 2,
        uniswapSlotIndex: 1,
        tokenType: 0,
        inputAmount: 10,
        minOutputAmount: 100,
        timeout: 10,
        signature: sig,
      }

      const transitionString: string = abiEncodeTransition(transition)
      const parsedTransition = parseTransitionFromABI(transitionString)

      parsedTransition.should.deep.equal(transition)
    })

    it('should encoded & decode Transfer Transition without throwing', async () => {
      const transition: TransferTransition = {
        stateRoot,
        senderSlotIndex: 2,
        recipientSlotIndex: 1,
        tokenType: 0,
        amount: 10,
        signature: sig,
      }

      const transitionString: string = abiEncodeTransition(transition)
      const parsedTransition = parseTransitionFromABI(transitionString)

      parsedTransition.should.deep.equal(transition)
    })

    it('should encoded & decode CreateAndTransfer Transition without throwing', async () => {
      const transition: CreateAndTransferTransition = {
        stateRoot,
        senderSlotIndex: 2,
        recipientSlotIndex: 1,
        tokenType: 0,
        amount: 10,
        signature: sig,
        createdAccountPubkey: AGGREGATOR_ADDRESS,
      }

      const transitionString: string = abiEncodeTransition(transition)
      const parsedTransition = parseTransitionFromABI(transitionString)

      parsedTransition.should.deep.equal(transition)
    })
  })
})
