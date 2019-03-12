import { account as Account } from 'eth-lib'
import { anyString, instance, mock, when } from 'ts-mockito'

import { ETHProvider } from '../../src/services'
import { EthereumAccount } from '../../src/services/models/eth'

const mockETHProvider = mock(ETHProvider)
const walletAccounts: { [key: string]: EthereumAccount } = {}
when(mockETHProvider.hasWalletAccount(anyString())).thenCall(
  (address: string) => {
    return address in walletAccounts
  }
)
when(mockETHProvider.addWalletAccount(anyString())).thenCall(
  (privateKey: string) => {
    const account = Account.fromPrivate(privateKey)
    walletAccounts[account.address] = account
  }
)
when(mockETHProvider.started).thenReturn(true)

const eth = instance(mockETHProvider)

export { mockETHProvider, eth }
