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
  runInDomain,
  MerkleTreeInclusionProof,
  ZERO,
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
  StateUpdate,
  StateInclusionProof,
  StateSnapshot,
  InclusionProof,
  StateMachineCapacityError,
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

  private lastOpenKey: BigNumber
  private readonly usedKeys: Set<string>
  private readonly addressesToKeys: Map<Address, BigNumber>
  private readonly maxAddresses: BigNumber

  private readonly tree: SparseMerkleTree
  private readonly lock: AsyncLock = new AsyncLock({
    domainReentrant: true,
  })

  public static async create(
    genesisState: State,
    db: DB,
    signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance(),
    swapFeeBasisPoints: number = 30,
    treeHeight: number = 32
  ): Promise<RollupStateMachine> {
    const stateMachine = new DefaultRollupStateMachine(
      db,
      signatureVerifier,
      swapFeeBasisPoints,
      treeHeight
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
    private swapFeeBasisPoints: number,
    treeHeight: number = 32
  ) {
    this.tree = new SparseMerkleTreeImpl(db, undefined, treeHeight)
    this.usedKeys = new Set<string>()
    this.lastOpenKey = ZERO
    this.addressesToKeys = new Map<Address, BigNumber>()
    this.maxAddresses = new BigNumber(Math.pow(2, this.tree.getHeight()) - 1)
  }

  public async getState(address: Address): Promise<StateSnapshot> {
    const [accountState, proof, stateRoot]: [
      Buffer,
      MerkleTreeInclusionProof,
      string
    ] = await this.lock.acquire(DefaultRollupStateMachine.lockKey, async () => {
      const key: BigNumber = this.getAddressKey(address)

      if (!!key) {
        const leaf: Buffer = await this.tree.getLeaf(key)
        if (!!leaf) {
          const merkleProof: MerkleTreeInclusionProof = await this.tree.getMerkleProof(
            key,
            leaf
          )
          return [leaf, merkleProof, merkleProof.rootHash.toString('hex')]
        }
      }

      return [
        undefined,
        undefined,
        (await this.tree.getRootHash()).toString('hex'),
      ]
    })

    let state: State
    let inclusionProof: InclusionProof
    if (!accountState) {
      state = undefined
      inclusionProof = undefined
    } else {
      state = this.deserializeState(address, accountState)
      inclusionProof = proof.siblings.map((x: Buffer) => x.toString('hex'))
    }

    return {
      address,
      state,
      stateRoot,
      inclusionProof,
    }
  }

  public async applyTransactions(
    transactions: SignedTransaction[]
  ): Promise<StateUpdate> {
    return runInDomain(undefined, async () => {
      return this.lock.acquire(DefaultRollupStateMachine.lockKey, async () => {
        const stateUpdates: StateUpdate[] = []

        for (const tx of transactions) {
          // TODO: How do we represent when some fail and some succeed, since the state will be partially updated?
          stateUpdates.push(await this.applyTransaction(tx))
        }

        const startRoot: string = stateUpdates[0].startRoot
        const endRoot: string = stateUpdates[stateUpdates.length - 1].endRoot
        const updatedState: State = {}
        const updatedStateInclusionProof: StateInclusionProof = {}
        for (const update of stateUpdates) {
          Object.assign(updatedState, update.updatedState)
          Object.assign(
            updatedStateInclusionProof,
            update.updatedStateInclusionProof
          )
        }

        return {
          transactions,
          startRoot,
          endRoot,
          updatedState,
          updatedStateInclusionProof,
        }
      })
    })
  }

  public async applyTransaction(
    signedTransaction: SignedTransaction
  ): Promise<StateUpdate> {
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
      const startRoot: string = (await this.tree.getRootHash()).toString('hex')
      const transaction: Transaction = signedTransaction.transaction
      let updatedState: State
      if (isTransferTransaction(transaction)) {
        updatedState = await this.applyTransfer(sender, transaction)
      } else if (isSwapTransaction(transaction)) {
        updatedState = await this.applySwap(sender, transaction)
      } else {
        throw new InvalidTransactionTypeError()
      }

      const updatedStateInclusionProof: StateInclusionProof = {}
      for (const key of Object.keys(updatedState)) {
        const proof: MerkleTreeInclusionProof = await this.tree.getMerkleProof(
          this.getAddressKey(key),
          this.serializeBalances(key, updatedState[key].balances)
        )
        updatedStateInclusionProof[key] = proof.siblings.map((p) =>
          p.toString('hex')
        )
      }

      const endRoot: string = (await this.tree.getRootHash()).toString('hex')

      return {
        transactions: [signedTransaction],
        startRoot,
        endRoot,
        updatedState,
        updatedStateInclusionProof,
      }
    })
  }

  private async getBalances(address: string): Promise<Balances> {
    const key: BigNumber = this.getAddressKey(address)

    if (!!key) {
      const leaf: Buffer = await this.tree.getLeaf(key)
      if (!!leaf) {
        return this.deserializeState(address, leaf)[address].balances
      }
    }
    return { uni: 0, pigi: 0 }
  }

  private async setAddressState(
    address: string,
    balances: Balances
  ): Promise<boolean> {
    const addressKey: BigNumber = this.getOrCreateAddressKey(address)
    const serializedBalances: Buffer = this.serializeBalances(address, balances)

    const result: boolean = await this.tree.update(
      addressKey,
      serializedBalances
    )

    return result
  }

  private async hasBalance(
    address: Address,
    tokenType: TokenType,
    balance: number
  ): Promise<boolean> {
    // Check that the account has more than some amount of pigi/uni
    const balances = await this.getBalances(address)
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
    return this.addressesToKeys.get(address)
  }

  private getOrCreateAddressKey(address: string): BigNumber {
    const existingKey: BigNumber = this.getAddressKey(address)
    if (!!existingKey) {
      return existingKey
    }

    let newKey: string = this.lastOpenKey.toString()
    while (this.usedKeys.has(newKey)) {
      this.lastOpenKey = this.lastOpenKey.add(ONE)
      if (this.lastOpenKey.gt(this.maxAddresses)) {
        throw new StateMachineCapacityError()
      }
      newKey = this.lastOpenKey.toString()
    }
    this.addressesToKeys.set(address, this.lastOpenKey)
    this.usedKeys.add(newKey)

    return this.addressesToKeys.get(address)
  }

  private serializeBalances(address: string, balances: Balances): Buffer {
    // TODO: Update these to deal with ABI encoding
    return objectToBuffer(this.getStateFromBalances(address, balances))
  }

  private deserializeState(address: string, state: Buffer): State {
    return deserializeBuffer(state)
  }

  private getStateFromBalances(address: string, balances: Balances): State {
    return {
      [address]: {
        balances,
      },
    }
  }
}
