/* External Imports */
import {
  DefaultWallet,
  DefaultWalletDB,
  BaseDB,
  WalletDB,
  sign,
  SignatureProvider,
} from '@pigi/core'

/* Internal Imports */
import { Address, Balances, TransactionReceipt, MockRollupClient } from '.'

const KEYSTORE_BUCKET = 0
const ROLLUP_BUCKET = 1

/*
 * The UnipigWallet class can be used to interact with the OVM and
 * all the L2s under it.
 */
export class UnipigWallet extends DefaultWallet {
  private db: BaseDB
  public rollup: MockRollupClient

  constructor(db: BaseDB, signatureProvider?: SignatureProvider) {
    // Set up the keystore db
    const keystoreBucket = db.bucket(Buffer.from([KEYSTORE_BUCKET]))
    const keystoreDB: WalletDB = new DefaultWalletDB(keystoreBucket)
    super(keystoreDB)

    // Set up the rollup client db
    const rollupBucket = db.bucket(Buffer.from([ROLLUP_BUCKET]))
    this.rollup = new MockRollupClient(rollupBucket, signatureProvider || this)

    // Save a reference to our db
    this.db = db
  }

  public async getBalances(account: Address): Promise<Balances> {
    // For now we only have one client so just get the rollup balance
    return this.rollup.getBalances(account)
  }
}
