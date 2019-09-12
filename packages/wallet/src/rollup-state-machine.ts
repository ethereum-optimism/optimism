/* External Imports */
import * as AsyncLock from 'async-lock'
import {
  DefaultSignatureVerifier,
  serializeObject,
  SignatureVerifier,
  DB,
  SparseMerkleTree,
  SparseMerkleTreeImpl,
  BigNumber,
  keccak256,
  objectToBuffer,
  deserializeBuffer,
  ONE,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  Balances,
  Swap,
  isSwapTransaction,
  Transfer,
  isTransferTransaction,
  Transaction,
  SignedTransaction,
  UNISWAP_ADDRESS,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  TokenType,
  State,
} from './index'
import {
  InsufficientBalanceError,
  InvalidTransactionTypeError,
  NegativeAmountError,
  RollupStateMachine,
  SlippageError,
} from './types'

export class DefaultRollupStateMachine implements RollupStateMachine {
  private static readonly lockKey: string = 'lock'

  private readonly tree: SparseMerkleTree
  private readonly lock: AsyncLock = new AsyncLock()

  public static async create(
    genesisState: State,
    db: DB,
    signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance(),
    swapFeeBasisPoints: number = 30
  ): Promise<RollupStateMachine> {
    const stateMachine = new DefaultRollupStateMachine(
      db,
      signatureVerifier,
      swapFeeBasisPoints
    )

    if (!!Object.keys(genesisState).length) {
      const promises: Array<Promise<boolean>> = []
      for (const key of Object.keys(genesisState)) {
        promises.push(
          stateMachine.setAddressState(key, genesisState[key].balances)
        )
      }
      await Promise.all(promises)
    }

    return stateMachine
  }

  private constructor(
    db: DB,
    private readonly signatureVerifier: SignatureVerifier,
    private swapFeeBasisPoints: number
  ) {
    this.tree = new SparseMerkleTreeImpl(db)
  }

  public async getBalances(account: Address): Promise<Balances> {
    const key: BigNumber = this.getAddressKey(account)
    const accountState: Buffer = await this.tree.getLeaf(key)

    let balances: Balances
    if (!accountState) {
      balances = {
        uni: 0,
        pigi: 0,
      }
    } else {
      balances = this.deserializeBalances(account, accountState)
    }
    return balances
  }

  public async applyTransaction(
    signedTransaction: SignedTransaction
  ): Promise<State> {
    let sender: Address

    try {
      sender = this.signatureVerifier.verifyMessage(
        serializeObject(signedTransaction.transaction),
        signedTransaction.signature
      )
    } catch (e) {
      throw e
    }

    return this.lock.acquire(DefaultRollupStateMachine.lockKey, async () => {
      const transaction: Transaction = signedTransaction.transaction
      if (isTransferTransaction(transaction)) {
        return this.applyTransfer(sender, transaction)
      } else if (isSwapTransaction(transaction)) {
        return this.applySwap(sender, transaction)
      }
      throw new InvalidTransactionTypeError()
    })
  }

  private async setAddressState(
    address: string,
    balances: Balances
  ): Promise<boolean> {
    const addressKey: BigNumber = this.getAddressKey(address)
    const serializedBalances: Buffer = this.serializeBalances(address, balances)

    const result: boolean = await this.tree.update(
      addressKey,
      serializedBalances
    )

    return result
  }

  private async hasBalance(
    account: Address,
    tokenType: TokenType,
    balance: number
  ): Promise<boolean> {
    // Check that the account has more than some amount of pigi/uni
    const balances = await this.getBalances(account)
    return tokenType in balances && balances[tokenType] >= balance
  }

  private async applyTransfer(
    sender: Address,
    transfer: Transfer
  ): Promise<State> {
    // Make sure the amount is above zero
    if (transfer.amount < 1) {
      throw new NegativeAmountError()
    }

    // Check that the sender has enough money
    if (!(await this.hasBalance(sender, transfer.tokenType, transfer.amount))) {
      throw new InsufficientBalanceError()
    }

    const senderBalances = await this.getBalances(sender)
    const recipientBalances = await this.getBalances(transfer.recipient)

    // Update the balances
    senderBalances[transfer.tokenType] -= transfer.amount
    recipientBalances[transfer.tokenType] += transfer.amount

    // TODO: use batch update
    await Promise.all([
      this.setAddressState(sender, senderBalances),
      this.setAddressState(transfer.recipient, recipientBalances),
    ])

    return {
      ...this.getStateFromBalances(sender, senderBalances),
      ...this.getStateFromBalances(transfer.recipient, recipientBalances),
    }
  }

  private async applySwap(sender: Address, swap: Swap): Promise<State> {
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
    return this.updateBalancesFromSwap(swap, sender)
  }

  private async updateBalancesFromSwap(
    swap: Swap,
    sender: Address
  ): Promise<State> {
    const uniswapBalances: Balances = await this.getBalances(UNISWAP_ADDRESS)
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

    const senderBalances: Balances = await this.getBalances(sender)
    // Calculate the new user & swap balances
    senderBalances[inputTokenType] -= swap.inputAmount
    senderBalances[outputTokenType] += outputAmount

    uniswapBalances[inputTokenType] += swap.inputAmount
    uniswapBalances[outputTokenType] = newOutputBalance

    // TODO: use batch update
    await Promise.all([
      this.setAddressState(sender, senderBalances),
      this.setAddressState(UNISWAP_ADDRESS, uniswapBalances),
    ])

    return {
      ...this.getStateFromBalances(sender, senderBalances),
      ...this.getStateFromBalances(UNISWAP_ADDRESS, uniswapBalances),
    }
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

  private getAddressKey(address: string): BigNumber {
    // TODO: This makes sure the key has all 0s for bits > tree height -- should be in merkle-tree.ts
    const andMask: BigNumber = ONE.shiftLeft(this.tree.getHeight() - 1).sub(ONE)
    return new BigNumber(keccak256(Buffer.from(address))).and(andMask)
  }

  private serializeBalances(address: string, balances: Balances): Buffer {
    return objectToBuffer(this.getStateFromBalances(address, balances))
  }

  private deserializeBalances(address: string, state: Buffer): Balances {
    const stateObj: State = deserializeBuffer(state)
    return stateObj[address].balances
  }

  private getStateFromBalances(address: string, balances: Balances): State {
    return {
      [address]: {
        balances,
      },
    }
  }
}
