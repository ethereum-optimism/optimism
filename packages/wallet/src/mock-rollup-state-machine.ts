/* External Imports */

/* Internal Imports */
import {
  Address,
  Balances,
  Swap,
  isSwapTransaction,
  Transfer,
  isTransferTransaction,
  Transaction,
  MockedSignature,
  SignedTransaction,
  TransactionReceipt,
  UNISWAP_ADDRESS,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  TokenType,
  State,
} from '.'

const DEFAULT_STORAGE = {
  balances: {
    uni: 0,
    pigi: 0,
  },
}

/*
 * Errors
 */
export class SlippageError extends Error {
  constructor() {
    super('Too much slippage in swap tx!')
  }
}

export class InsufficientBalanceError extends Error {
  constructor() {
    super('Insufficient balance for transfer or swap!')
  }
}

export class NegativeAmountError extends Error {
  constructor() {
    super('Amounts transferred or swapped cannot be negative!')
  }
}

export class InvalidTransactionTypeError extends Error {
  constructor() {
    super('Invalid transaction type!')
  }
}

export class MockRollupStateMachine {
  public state: State

  constructor(genesisState: State, private swapFeeBasisPoints: number = 30) {
    this.state = genesisState
  }

  public getBalances(account: Address) {
    return this._getBalances(account, false)
  }

  public getUniswapBalances(): Balances {
    return this._getBalances(UNISWAP_ADDRESS)
  }

  private _getBalances(
    account: Address,
    createIfAbsent: boolean = true
  ): Balances {
    if (!(account in this.state)) {
      const balances = {
        uni: 0,
        pigi: 0,
      }

      if (!createIfAbsent) {
        return balances
      }

      this.state[account] = { balances }
    }
    return this.state[account].balances
  }

  private ecdsaRecover(signature: MockedSignature): Address {
    // TODO: Move this out of this class and instead put in keystore
    return signature
  }

  public applyTransaction(
    signedTransaction: SignedTransaction
  ): TransactionReceipt {
    const sender: Address = signedTransaction.signature
    const transaction: Transaction = signedTransaction.transaction
    if (isTransferTransaction(transaction)) {
      return this.applyTransfer(sender, transaction)
    } else if (isSwapTransaction(transaction)) {
      return this.applySwap(sender, transaction)
    }
    throw new InvalidTransactionTypeError()
  }

  private getTxReceipt(stateUpdate: any): TransactionReceipt {
    return {
      aggregatorSignature: 'MOCKED',
      stateUpdate,
    }
  }

  private hasBalance(account: Address, tokenType: TokenType, balance: number) {
    // Check that the account has more than some amount of pigi/uni
    const balances = this._getBalances(account, false)
    return balances[tokenType] >= balance
  }

  private applyTransfer(
    sender: Address,
    transfer: Transfer
  ): TransactionReceipt {
    // Make sure the amount is above zero
    if (transfer.amount < 1) {
      throw new NegativeAmountError()
    }
    // Check that the sender has enough money
    if (!this.hasBalance(sender, transfer.tokenType, transfer.amount)) {
      throw new InsufficientBalanceError()
    }

    // Update the balances
    this._getBalances(sender)[transfer.tokenType] -= transfer.amount
    this._getBalances(transfer.recipient)[transfer.tokenType] += transfer.amount

    return this.getTxReceipt({
      sender: this.state[sender],
      recipient: this.state[transfer.recipient],
    })
  }

  private applySwap(sender: Address, swap: Swap): TransactionReceipt {
    // Make sure the amount is above zero
    if (swap.inputAmount < 1) {
      throw new NegativeAmountError()
    }
    // Check that the sender has enough money
    if (!this.hasBalance(sender, swap.tokenType, swap.inputAmount)) {
      throw new InsufficientBalanceError()
    }
    // Check that we'll have ample time to include the swap
    // TODO

    // Set the post swap balances
    this.updateBalancesFromSwap(swap, sender)

    // Return a succssful swap!
    return this.getTxReceipt({
      sender: this.state[sender],
      uniswap: this.state[UNISWAP_ADDRESS],
    })
  }

  private updateBalancesFromSwap(swap: Swap, sender: Address): void {
    const uniswapBalances: Balances = this.getUniswapBalances()
    // First let's figure out which token types are input & output
    const inputTokenType = swap.tokenType
    const outputTokenType =
      swap.tokenType === UNI_TOKEN_TYPE ? PIGI_TOKEN_TYPE : UNI_TOKEN_TYPE
    // Next let's calculate the invariant
    const invariant = uniswapBalances.uni * uniswapBalances.pigi
    // Now calculate the total input tokens
    const totalInput =
      this.assessSwapFee(swap.inputAmount) + uniswapBalances[inputTokenType]
    const newOutputBalance = Math.ceil(invariant / totalInput)
    const outputAmount = uniswapBalances[outputTokenType] - newOutputBalance
    // Let's make sure the output amount is above the minimum
    if (outputAmount < swap.minOutputAmount) {
      throw new SlippageError()
    }

    const userBalances: Balances = this._getBalances(sender)
    // Calculate the new user & swap balances
    userBalances[inputTokenType] -= swap.inputAmount
    userBalances[outputTokenType] += outputAmount

    uniswapBalances[inputTokenType] += swap.inputAmount
    uniswapBalances[outputTokenType] = newOutputBalance
  }

  /**
   * Assesses the fee charged for a swap.
   *
   * @param amountBeforeFee The amount of the swap
   * @return the amount, accounting for the fee
   */
  private assessSwapFee(amountBeforeFee: number): number {
    if (this.swapFeeBasisPoints === 0) {
      return amountBeforeFee
    }
    return amountBeforeFee * ((10_000.0 - this.swapFeeBasisPoints) / 10_000.0)
  }
}
