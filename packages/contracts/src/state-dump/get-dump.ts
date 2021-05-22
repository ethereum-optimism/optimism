/* External Imports */
import * as path from 'path'

export interface StorageDump {
  [key: string]: string
}

export interface StateDump {
  accounts: {
    [name: string]: {
      address: string
      code: string
      codeHash: string
      storage: StorageDump
      abi: any
    }
  }
}

export const getLatestStateDump = (): StateDump => {
  return require(path.join(__dirname, '../dumps', `state-dump.latest.json`))
}
