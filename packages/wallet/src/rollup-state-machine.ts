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
  NULL_ADDRESS,
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

const log = getLogger('rollup-state-machine')

/**
 * A Tree-backed Rollup State Machine, facilitating state transitions for
 * swaps and transactions for Uniswap.
 */
export class DefaultRollupStateMachine implements RollupStateMachine {
  public static readonly ROOT_KEY: Buffer = Buffer.from('state_machine_root')
  public static readonly LAST_OPEN_KEY: Buffer = Buffer.from('last_open_key')
  public static readonly ADDRESS_TO_KEYS_COUNT_KEY: Buffer = Buffer.from(
    'address_to_keys_count'
  )

  private static readonly lockKey: string = 'lock'

  private lastOpenKey: BigNumber
  private usedKeys: Set<string>
  private addressesToKeys: Map<Address, BigNumber>
  private maxAddresses: BigNumber
  private tree: SparseMerkleTree
  private readonly lock: AsyncLock

  /**
   * Creates and initializes a DefaultRollupStateMachine.
   *
   * @param genesisState The genesis state to set
   * @param db The DB to use
   * @param signatureVerifier The signature verifier to use
   * @param swapFeeBasisPoints The fee for swapping, in basis points
   * @param treeHeight The height of the tree to use for underlying storage
   * @returns The constructed and initialized RollupStateMachine
   */
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

    const previousStateExists = await stateMachine.init()

    if (!previousStateExists && !!genesisState.length) {
      const promises: Array<Promise<boolean>> = []
      for (const state of genesisState) {
        promises.push(
          stateMachine.setAddressState(state.pubkey, state.balances)
        )
      }
      await Promise.all(promises)
    }

    return stateMachine
  }

  private constructor(
    private readonly db: DB,
    private readonly signatureVerifier: SignatureVerifier,
    private readonly swapFeeBasisPoints: number,
    private readonly treeHeight: number = 32
  ) {
    this.maxAddresses = new BigNumber(Math.pow(2, this.treeHeight) - 1)
    this.lock = new AsyncLock({
      domainReentrant: true,
    })
  }

  /**
   * Initializes this RollupStateMachine, reading stored state from the DB
   * and populating local variables from saved state if there is any.
   *
   * @returns True if there was existing state, false otherwise.
   */
  private async init(): Promise<boolean> {
    const storedRoot: Buffer = await this.db.get(
      DefaultRollupStateMachine.ROOT_KEY
    )

    this.tree = await SparseMerkleTreeImpl.create(
      this.db,
      storedRoot,
      this.treeHeight
    )
    this.addressesToKeys = new Map<Address, BigNumber>()
    this.usedKeys = new Set<string>()
    this.lastOpenKey = ZERO

    if (!storedRoot) {
      log.info(
        `No existing state root found, starting RollupStateMachine fresh`
      )
      return false
    }

    const [lastKeyBuffer, addressToKeyCountBuffer] = await Promise.all([
      this.db.get(DefaultRollupStateMachine.LAST_OPEN_KEY),
      this.db.get(DefaultRollupStateMachine.ADDRESS_TO_KEYS_COUNT_KEY),
    ])

    this.lastOpenKey = new BigNumber(lastKeyBuffer)
    const addressCount = parseInt(addressToKeyCountBuffer.toString(), 10)

    log.info(
      `RollupStateMachine found root ${storedRoot.toString(
        'hex'
      )}, last open key: ${this.lastOpenKey.toString(
        10
      )}, and address key count: ${addressCount}. Initializing with this state.`
    )

    const addressPromises: Array<Promise<Buffer>> = []
    for (let i = 0; i < addressCount; i++) {
      addressPromises.push(
        this.db.get(DefaultRollupStateMachine.getAddressMapDBKey(i))
      )
    }

    const addressToKeysBuffers: Buffer[] = await Promise.all(addressPromises)
    for (const addressKeyBuf of addressToKeysBuffers) {
      const addressAndKey: any[] = DefaultRollupStateMachine.deserializeAddressToKeyFromDB(
        addressKeyBuf
      )
      this.addressesToKeys.set(addressAndKey[0], addressAndKey[1])
    }

    for (const key of this.addressesToKeys.values()) {
      this.usedKeys.add(key.toString())
      if (key.gt(this.lastOpenKey)) {
        this.lastOpenKey = key
      }
    }

    return true
  }

  /**
   * Gets the state associated with the provided address.
   *
   * @param address The address for which state will be fetched
   * @returns The snapshot of the address's state
   */
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
      state = DefaultRollupStateMachine.deserializeState(accountState)
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

  /**
   * Applies the provided transactions and returns the resulting state updates.
   *
   * @param transactions The transactions to apply
   * @returns The updated state
   */
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

  /**
   * Applies the provided SignedTransaction, returning the resulting statue update.
   *
   * @param signedTransaction The transaction to apply
   * @returns The updated state
   */
  public async applyTransaction(
    signedTransaction: SignedTransaction
  ): Promise<StateUpdate> {
    let signer: Address

    log.debug(`Validating Signature: ${JSON.stringify(signedTransaction)}`)
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

      const root: Buffer = await this.tree.getRootHash()
      await this.db.put(DefaultRollupStateMachine.ROOT_KEY, root)

      const senderState: State = updatedStates[0]
      const receiverState: State = updatedStates[1]

      stateUpdate['senderSlotIndex'] = this.getAddressKey(
        transaction.sender
      ).toNumber()
      stateUpdate['senderState'] = senderState
      stateUpdate['receiverState'] = receiverState

      const inclusionProof = async (state: State): Promise<InclusionProof> => {
        const proof: MerkleTreeInclusionProof = await this.tree.getMerkleProof(
          this.getAddressKey(state.pubkey),
          DefaultRollupStateMachine.serializeBalances(
            state.pubkey,
            state.balances
          )
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
    const lockedRoot = await this.lock.acquire(
      DefaultRollupStateMachine.lockKey,
      async () => {
        return this.tree.getRootHash()
      }
    )
    return lockedRoot
  }

  public getNextNewAccountSlot(): number {
    return this.lastOpenKey.toNumber() + 1
  }

  public async getSnapshotFromSlot(key: number): Promise<StateSnapshot> {
    const [accountState, proof, stateRoot]: [
      Buffer,
      MerkleTreeInclusionProof,
      string
    ] = await this.lock.acquire(DefaultRollupStateMachine.lockKey, async () => {
      const bigKey: BigNumber = new BigNumber(key)
      let leaf: Buffer = await this.tree.getLeaf(bigKey)

      if (!leaf) {
        // if we didn't get the leaf it must be empty
        leaf = SparseMerkleTreeImpl.emptyBuffer
      }

      const merkleProof: MerkleTreeInclusionProof = await this.tree.getMerkleProof(
        new BigNumber(key),
        leaf
      )
      return [leaf, merkleProof, merkleProof.rootHash.toString('hex')]
    })

    let state: State
    let inclusionProof: InclusionProof
    state =
      !!accountState && !accountState.equals(SparseMerkleTreeImpl.emptyBuffer)
        ? DefaultRollupStateMachine.deserializeState(accountState)
        : {
            pubkey: NULL_ADDRESS,
            balances: { [UNI_TOKEN_TYPE]: 0, [PIGI_TOKEN_TYPE]: 0 },
          }
    inclusionProof = proof.siblings.map((x: Buffer) => x.toString('hex'))

    return {
      slotIndex: key,
      state,
      stateRoot,
      inclusionProof,
    }
  }

  /**
   * Gets the balances for the provided address.
   *
   * @param address The address in question
   * @returns The provided address's balances
   */
  private async getBalances(address: string): Promise<Balances> {
    const key: BigNumber = this.getAddressKey(address)

    if (!!key) {
      const leaf: Buffer = await this.tree.getLeaf(key)
      if (!!leaf) {
        return DefaultRollupStateMachine.deserializeState(leaf).balances
      }
    }
    return { [UNI_TOKEN_TYPE]: 0, [PIGI_TOKEN_TYPE]: 0 }
  }

  /**
   * Sets the state for the provided address to be the provided balances.
   *
   * @param address The address to set
   * @param balances The balances state to set
   * @returns True if the updates succeeded, false otherwise
   */
  private async setAddressState(
    address: string,
    balances: Balances
  ): Promise<boolean> {
    const addressKey: BigNumber = await this.getOrCreateAddressKey(address)
    const serializedBalances: Buffer = DefaultRollupStateMachine.serializeBalances(
      address,
      balances
    )

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

  /**
   * Determines whether the provided address has the provided balance of the
   * provided token type.
   *
   * @param address The address in question
   * @param tokenType The token type
   * @param balance The balance
   * @returns True if so, false otherwise
   */
  private async hasBalance(
    address: Address,
    tokenType: TokenType,
    balance: number
  ): Promise<boolean> {
    // Check that the account has more than some amount of pigi/uni
    const balances = await this.getBalances(address)
    return tokenType in balances && balances[tokenType] >= balance
  }

  /**
   * Applies the provided Transfer transaction, returning the updated state.
   *
   * @param transfer The Transfer in question
   * @returns The updated balances
   */
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
      DefaultRollupStateMachine.getStateFromBalances(
        transfer.sender,
        senderBalances
      ),
      DefaultRollupStateMachine.getStateFromBalances(
        transfer.recipient,
        recipientBalances
      ),
    ]
  }

  /**
   * Applies the provided Swap transaction, returning the updated state.
   *
   * @param sender The sender of the Swap
   * @param swap The swap
   * @returns The updated balances
   */
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

  /**
   * Sets and returnsthe new balances for Uniswap and the provided sender
   * from the provided Swap.
   *
   * @param swap The swap in question
   * @param sender The sender of the swap transaction
   * @returns The resulting state
   */
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
      DefaultRollupStateMachine.getStateFromBalances(sender, senderBalances),
      DefaultRollupStateMachine.getStateFromBalances(
        UNISWAP_ADDRESS,
        uniswapBalances
      ),
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

  private async getOrCreateAddressKey(address: string): Promise<BigNumber> {
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

    // Order of updates matters here, so can't parallelize
    await this.db.put(
      DefaultRollupStateMachine.getAddressMapDBKey(
        this.addressesToKeys.size - 1
      ),
      DefaultRollupStateMachine.serializeAddressToKeyForDB(
        address,
        this.lastOpenKey
      )
    )
    await Promise.all([
      this.db.put(
        DefaultRollupStateMachine.ADDRESS_TO_KEYS_COUNT_KEY,
        Buffer.from(this.addressesToKeys.size.toString(10))
      ),
      this.db.put(
        DefaultRollupStateMachine.LAST_OPEN_KEY,
        this.lastOpenKey.toBuffer()
      ),
    ])

    return this.addressesToKeys.get(address)
  }

  /********************
   * STATIC UTILITIES *
   ********************/

  public static serializeBalances(address: string, balances: Balances): Buffer {
    return Buffer.from(
      abiEncodeState(
        DefaultRollupStateMachine.getStateFromBalances(address, balances)
      )
    )
  }

  public static deserializeState(state: Buffer): State {
    return parseStateFromABI(state.toString())
  }

  public static getStateFromBalances(
    pubKey: string,
    balances: Balances
  ): State {
    return {
      pubkey: pubKey,
      balances,
    }
  }

  public static getAddressMapDBKey(index: number): Buffer {
    return Buffer.from(`ADDR_IDX_${index}`)
  }

  public static serializeAddressToKeyForDB(
    address: Address,
    key: BigNumber
  ): Buffer {
    return Buffer.from(JSON.stringify([address, key.toString()]))
  }

  public static deserializeAddressToKeyFromDB(buf: Buffer): any[] {
    const parsed: any[] = JSON.parse(buf.toString())
    return [parsed[0], new BigNumber(parsed[1])]
  }

  /***********
   * GETTERS *
   ***********/
  public getLastOpenKey(): BigNumber {
    return this.lastOpenKey.clone()
  }

  public getUsedKeys(): Set<string> {
    return new Set<string>(this.usedKeys)
  }

  public getAddressesToKeys(): Map<Address, BigNumber> {
    return new Map<Address, BigNumber>(this.addressesToKeys)
  }
}
