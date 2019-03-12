import { anyString, anything, instance, mock, when } from 'ts-mockito'

import { WalletDB } from '../../../src/services'
import { EthereumAccount } from '../../../src/services/models/eth'

const mockWalletDB = mock(WalletDB)
const accounts: { [key: string]: EthereumAccount } = {}
when(mockWalletDB.addAccount(anything())).thenCall(
  (account: EthereumAccount) => {
    accounts[account.address] = account
  }
)
when(mockWalletDB.getAccount(anyString())).thenCall((address: string) => {
  return accounts[address]
})
when(mockWalletDB.getAccounts()).thenCall(() => {
  const arr = []
  for (const account of Object.keys(accounts)) {
    arr.push(account)
  }
  return arr
})
when(mockWalletDB.started).thenReturn(true)

const walletdb = instance(mockWalletDB)

export { mockWalletDB, walletdb }
