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
  ONE,
  runInDomain,
  MerkleTreeInclusionProof,
  ZERO,
  getLogger,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  Balances,
  Swap,
  isSwapTransaction,
  Transfer,
  isTransferTransaction,
  RollupTransaction,
  SignedTransaction,
  UNISWAP_ADDRESS,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  TokenType,
  State,
  StateUpdate,
  StateSnapshot,
  InclusionProof,
  StateMachineCapacityError,
  SignatureError,
  AGGREGATOR_ADDRESS,
  abiEncodeTransaction,
  abiEncodeState,
  parseStateFromABI,
  NON_EXISTENT_SLOT_INDEX,
} from './index'
import {
  InsufficientBalanceError,
  InvalidTransactionTypeError,
  NegativeAmountError,
  RollupStateMachine,
  SlippageError,
} from './types'

const log = getLogger('rollup-aggregator')

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
    genesisState: State[],
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

    if (!!genesisState.length) {
      const promises: Array<Promise<boolean>> = []
      for (const state of genesisState) {
        promises.push(
          stateMachine.setAddressState(state.pubKey, state.balances)
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
    let slotIndex: number
    if (!accountState) {
      state = undefined
      inclusionProof = undefined
      slotIndex = NON_EXISTENT_SLOT_INDEX
    } else {
      state = this.deserializeState(accountState)
      inclusionProof = proof.siblings.map((x: Buffer) => x.toString('hex'))
      slotIndex = this.getAddressKey(address).toNumber()
    }

    return {
      slotIndex,
      state,
      stateRoot,
      inclusionProof,
    }
  }

  public async applyTransactions(
    transactions: SignedTransaction[]
  ): Promise<StateUpdate[]> {
    return runInDomain(undefined, async () => {
      return this.lock.acquire(DefaultRollupStateMachine.lockKey, async () => {
        const stateUpdates: StateUpdate[] = []

        for (const tx of transactions) {
          // TODO: How do we represent when some fail and some succeed, since the state will be partially updated?
          stateUpdates.push(await this.applyTransaction(tx))
        }

        return stateUpdates
      })
    })
  }

  public async applyTransaction(
    signedTransaction: SignedTransaction
  ): Promise<StateUpdate> {
    let signer: Address

    signer = this.signatureVerifier.verifyMessage(
      abiEncodeTransaction(signedTransaction.transaction),
      signedTransaction.signature
    )
    if (
      signer !== signedTransaction.transaction.sender &&
      signer !== AGGREGATOR_ADDRESS
    ) {
      log.info(
        `Received transaction with invalid signature: ${serializeObject(
          signedTransaction
        )}, which recovered a signer of ${signer}`
      )
      throw new SignatureError()
    }

    return this.lock.acquire(DefaultRollupStateMachine.lockKey, async () => {
      const stateUpdate = { transaction: signedTransaction }
      const transaction: RollupTransaction = signedTransaction.transaction
      let updatedStates: State[]
      if (isTransferTransaction(transaction)) {
        stateUpdate['receiverCreated'] = !this.getAddressKey(
          transaction.recipient
        )
        updatedStates = await this.applyTransfer(transaction)
        stateUpdate['receiverSlotIndex'] = this.getAddressKey(
          transaction.recipient
        ).toNumber()
      } else if (isSwapTransaction(transaction)) {
        updatedStates = await this.applySwap(signer, transaction)
        stateUpdate['receiverCreated'] = false
        stateUpdate['receiverSlotIndex'] = this.getAddressKey(
          UNISWAP_ADDRESS
        ).toNumber()
      } else {
        throw new InvalidTransactionTypeError()
      }
      const senderState: State = updatedStates[0]
      const receiverState: State = updatedStates[1]

      stateUpdate['senderSlotIndex'] = this.getAddressKey(
        transaction.sender
      ).toNumber()
      stateUpdate['senderState'] = senderState
      stateUpdate['receiverState'] = receiverState

      const inclusionProof = async (state: State): Promise<InclusionProof> => {
        const proof: MerkleTreeInclusionProof = await this.tree.getMerkleProof(
          this.getAddressKey(state.pubKey),
          this.serializeBalances(state.pubKey, state.balances)
        )
        return proof.siblings.map((p) => p.toString('hex'))
      }
      ;[
        stateUpdate['senderStateInclusionProof'],
        stateUpdate['receiverStateInclusionProof'],
      ] = await Promise.all([
        inclusionProof(senderState),
        inclusionProof(receiverState),
      ])

      stateUpdate['stateRoot'] = (await this.tree.getRootHash()).toString('hex')
      return stateUpdate
    })
  }

  public async getStateRoot(): Promise<Buffer> {
    return this.tree.getRootHash()
  }

  public getNextNewAccountSlot(): number {
    return this.lastOpenKey.toNumber()
  }

  public async getSnapshotFromSlot(key: number): Promise<StateSnapshot> {
    const [accountState, proof, stateRoot]: [
      Buffer,
      MerkleTreeInclusionProof,
      string
    ] = await this.lock.acquire(DefaultRollupStateMachine.lockKey, async () => {
      let leaf: Buffer = await this.tree.getLeaf(new BigNumber(key))
      if (!leaf) {leaf = Buffer.alloc(32).fill('\x00')}
      // console.log('leaf is: ')
      // console.log(leaf)

      const merkleProof: MerkleTreeInclusionProof = await this.tree.getMerkleProof(
        new BigNumber(key),
        leaf
      )
      // console.log('merkleproof is: ')
      // console.log(merkleProof)
      return [leaf, merkleProof, merkleProof.rootHash.toString('hex')]
    })

    let state: State
    let inclusionProof: InclusionProof
    state = accountState ? this.deserializeState(accountState) : undefined
    inclusionProof = proof.siblings.map((x: Buffer) => x.toString('hex'))

    return {
      slotIndex: key,
      state,
      stateRoot,
      inclusionProof,
    }
  }

  private async getBalances(address: string): Promise<Balances> {
    const key: BigNumber = this.getAddressKey(address)

    if (!!key) {
      const leaf: Buffer = await this.tree.getLeaf(key)
      if (!!leaf) {
        return this.deserializeState(leaf).balances
      }
    }
    return { [UNI_TOKEN_TYPE]: 0, [PIGI_TOKEN_TYPE]: 0 }
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
    if (!result) {
      log.error(
        `ERROR UPDATING TREE, address: [${address}], key: [${addressKey}], balances: [${serializeObject(
          balances
        )}]`
      )
    } else {
      log.debug(
        `${address} with key ${addressKey} balance updated to ${serializeObject(
          balances
        )}`
      )
    }

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

  private async applyTransfer(transfer: Transfer): Promise<State[]> {
    // Make sure the amount is above zero
    if (transfer.amount < 1) {
      throw new NegativeAmountError()
    }

    // Check that the sender has enough money
    if (
      !(await this.hasBalance(
        transfer.sender,
        transfer.tokenType,
        transfer.amount
      ))
    ) {
      throw new InsufficientBalanceError()
    }

    const senderBalances = await this.getBalances(transfer.sender)
    const recipientBalances = await this.getBalances(transfer.recipient)

    // Update the balances
    senderBalances[transfer.tokenType] -= transfer.amount
    recipientBalances[transfer.tokenType] += transfer.amount

    // TODO: use batch update
    await Promise.all([
      this.setAddressState(transfer.sender, senderBalances),
      this.setAddressState(transfer.recipient, recipientBalances),
    ])

    return [
      this.getStateFromBalances(transfer.sender, senderBalances),
      this.getStateFromBalances(transfer.recipient, recipientBalances),
    ]
  }

  private async applySwap(sender: Address, swap: Swap): Promise<State[]> {
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
  ): Promise<State[]> {
    const uniswapBalances: Balances = await this.getBalances(UNISWAP_ADDRESS)
    // First let's figure out which token types are input & output
    const inputTokenType = swap.tokenType
    const outputTokenType =
      swap.tokenType === UNI_TOKEN_TYPE ? PIGI_TOKEN_TYPE : UNI_TOKEN_TYPE
    // Next let's calculate the invariant
    const invariant =
      uniswapBalances[UNI_TOKEN_TYPE] * uniswapBalances[PIGI_TOKEN_TYPE]
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

    return [
      this.getStateFromBalances(sender, senderBalances),
      this.getStateFromBalances(UNISWAP_ADDRESS, uniswapBalances),
    ]
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
    return Buffer.from(
      abiEncodeState(this.getStateFromBalances(address, balances))
    )
  }

  private deserializeState(state: Buffer): State {
    return parseStateFromABI(state.toString())
  }

  private getStateFromBalances(pubKey: string, balances: Balances): State {
    return {
      pubKey,
      balances,
    }
  }
}
