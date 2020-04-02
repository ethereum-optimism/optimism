import {DB} from '@eth-optimism/core-db'
import {L1ToL2Transaction, L1ToL2TransactionReceiver} from '@eth-optimism/rollup-core'

export class FullNodeL1ToL2TransactionReceiver implements L1ToL2TransactionReceiver {



  constructor(db: DB) {

  }


  public handleL1ToL2Transaction(transaction: L1ToL2Transaction): Promise<void> {

  }

}