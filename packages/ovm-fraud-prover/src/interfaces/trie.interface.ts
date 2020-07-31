/* External Imports */
import { BaseTrie } from 'merkle-patricia-tree'

export interface WorldState {
  stateTrie: BaseTrie
  accountTries: {
    [address: string]: BaseTrie
  }
}
