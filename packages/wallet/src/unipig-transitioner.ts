/* External Imports */
import {
  DefaultWallet,
  DefaultWalletDB,
  WalletDB,
  SignatureProvider,
  DB,
  SignatureVerifier,
  DefaultSignatureVerifier,
  getLogger,
  serializeObject,
  logError,
  SignedByDBInterface,
  SignedByDB,
  SignedByDecider,
  MerkleInclusionProofDecider,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  AGGREGATOR_ADDRESS,
  Balances,
  DefaultRollupStateSolver,
  EMPTY_AGGREGATOR_SIGNATURE,
  FaucetRequest,
  isFaucetTransaction,
  RollupClient,
  RollupStateSolver,
  RollupTransaction,
  SignatureError,
  SignedStateReceipt,
  StateReceipt,
  Swap,
  TokenType,
  Transfer,
  UNISWAP_ADDRESS,
} from '.'

const log = getLogger('unipig-transitioner')

interface KnownState {
  [pubKey: string]: StateReceipt
}

/*
 * The UnipigTransitioner class can be used to interact with the OVM and
 * all the L2s under it.
 */
export class UnipigTransitioner extends DefaultWallet {
  private db: DB
  private rollupClient: RollupClient
  private stateSolver: RollupStateSolver
  private knownState: KnownState

  public static new(db: DB, myAddress: string): UnipigTransitioner {
    const signedByDB: SignedByDBInterface = new SignedByDB(db)
    const decider: SignedByDecider = new SignedByDecider(
      signedByDB,
      Buffer.from(myAddress)
    )
    const stateSolver: RollupStateSolver = new DefaultRollupStateSolver(
      signedByDB,
      decider,
      new MerkleInclusionProofDecider()
    )

    const rollupclient: RollupClient = new RollupClient(db)

    return new UnipigTransitioner(db, stateSolver, rollupclient)
  }

  constructor(
    db: DB,
    stateSolver: RollupStateSolver,
    rollupClient: RollupClient,
    signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance(),
    signatureProvider?: SignatureProvider
  ) {
    // Set up the keystore db
    const keystoreDB: WalletDB = new DefaultWalletDB(db)
    super(keystoreDB)

    this.rollupClient = rollupClient

    // Save a reference to our db
    this.db = db
    this.stateSolver = stateSolver
    this.knownState = {}
  }

  public async getUniswapBalances(): Promise<Balances> {
    return this.getBalances(UNISWAP_ADDRESS)
  }

  public async getUniswapState(): Promise<StateReceipt> {
    return this.getState(UNISWAP_ADDRESS)
  }

  public async getBalances(account: Address): Promise<Balances> {
    const stateReceipt: StateReceipt = await this.getState(account)
    return !!stateReceipt ? stateReceipt.state.balances : undefined
  }

  public async getState(account: Address): Promise<StateReceipt> {
    // For now we only have one client so just get the rollup balance
    const signedState: SignedStateReceipt = await this.rollupClient.getState(
      account
    )

    await this.stateSolver.storeSignedStateReceipt(signedState)

    // If valid, update known state
    if (
      (account in this.knownState &&
        signedState.signature === EMPTY_AGGREGATOR_SIGNATURE) ||
      (await this.stateSolver.isStateReceiptProvablyValid(
        signedState.stateReceipt,
        AGGREGATOR_ADDRESS
      ))
    ) {
      this.knownState[account] = signedState.stateReceipt
    }

    return account in this.knownState ? this.knownState[account] : undefined
  }

  public async send(
    tokenType: TokenType,
    from: Address,
    to: Address,
    amount: number
  ): Promise<void> {
    const transaction: Transfer = {
      sender: from,
      recipient: to,
      tokenType,
      amount,
    }
    await this.submitTransaction(transaction)
  }

  public async swap(
    tokenType: TokenType,
    from: Address,
    inputAmount: number,
    minOutputAmount: number,
    timeoutMillis: number
  ): Promise<void> {
    const transaction: Swap = {
      sender: from,
      tokenType,
      inputAmount,
      minOutputAmount,
      timeout: timeoutMillis,
    }
    await this.submitTransaction(transaction)
  }

  public async requestFaucetFunds(
    forAddress: Address,
    amount: number = 10
  ): Promise<void> {
    const faucetRequest: FaucetRequest = {
      sender: forAddress,
      amount,
    }
    await this.submitTransaction(faucetRequest)
  }

  private async submitTransaction(
    transaction: RollupTransaction
  ): Promise<void> {
    let receipts: SignedStateReceipt[] = []
    try {
      if (isFaucetTransaction(transaction)) {
        receipts = [
          await this.rollupClient.requestFaucetFunds(transaction, this),
        ]
      } else {
        receipts = await this.rollupClient.sendTransaction(transaction, this)
      }
    } catch (e) {
      if (e instanceof SignatureError) {
        logError(
          log,
          `Transaction was sent and a response was received, indicating it probably went through, but the signature on the response was invalid, so we cannot be sure. Tx: ${serializeObject(
            transaction
          )}`,
          e
        )
      } else {
        logError(
          log,
          `An error occurred sending transaction: Tx: ${serializeObject(
            transaction
          )}.`,
          e
        )
      }
      throw e
    }

    try {
      await Promise.all(
        receipts.map((r) => this.stateSolver.storeSignedStateReceipt(r))
      )
    } catch (e) {
      logError(
        log,
        `An error persisting transaction receipt for Tx: ${serializeObject(
          transaction
        )}.`,
        e
      )
    }
  }
}
