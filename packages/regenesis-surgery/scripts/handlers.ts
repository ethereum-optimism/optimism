import { ethers } from 'ethers'
import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'
import { OLD_ETH_ADDRESS } from './constants'
import { Account, AccountType, SurgeryDataSources } from './types'
import { findAccount } from './utils'

export const handlers: {
  [key in AccountType]: (
    account: Account,
    data: SurgeryDataSources
  ) => Account | Promise<Account>
} = {
  [AccountType.EOA]: (account) => {
    return {
      address: account.address,
      nonce: account.nonce,
      balance: account.balance,
      codeHash: KECCAK256_NULL_S,
      root: KECCAK256_RLP_S,
    }
  },
  [AccountType.PRECOMPILE]: (account) => {
    return account
  },
  [AccountType.PREDEPLOY_DEAD]: () => {
    return undefined // delete the account
  },
  [AccountType.PREDEPLOY_WIPE]: (account, data) => {
    const genesisAccount = findAccount(data.genesis, account.address)
    return {
      ...account,
      code: genesisAccount.code,
      codeHash: genesisAccount.codeHash,
      storage: genesisAccount.storage,
    }
  },
  [AccountType.PREDEPLOY_NO_WIPE]: (account, data) => {
    const genesisAccount = findAccount(data.genesis, account.address)
    return {
      ...account,
      code: genesisAccount.code,
      codeHash: genesisAccount.codeHash,
      storage: {
        ...account.storage,
        ...genesisAccount.storage,
      },
    }
  },
  [AccountType.PREDEPLOY_ETH]: (account, data) => {
    const genesisAccount = findAccount(data.genesis, account.address)
    const oldEthAccount = findAccount(data.dump, OLD_ETH_ADDRESS)
    return {
      ...account,
      code: genesisAccount.code,
      codeHash: genesisAccount.codeHash,
      storage: {
        ...oldEthAccount.storage,
        ...genesisAccount.storage,
      },
    }
  },
  [AccountType.PREDEPLOY_WETH]: (account, data) => {
    // TODO
    throw new Error('Not implemented')
  },
  [AccountType.UNISWAP_V3_FACTORY]: () => {
    // TODO
    // Transfer the owner slot
    // Transfer the feeAmountTickSpacing slot
    // Transfer the getPool slot
    throw new Error('Not implemented')
  },
  [AccountType.UNISWAP_V3_NFPM]: () => {
    // TODO
    // Transfer the _poolIds slot
    throw new Error('Not implemented')
  },
  [AccountType.UNISWAP_V3_POOL]: async (account, data) => {
    const poolData = data.pools.find((pool) => {
      return pool.oldAddress === account.address
    })
    const poolCode = await data.l1TestnetProvider.getCode(poolData.newAddress)
    return {
      ...account,
      address: poolData.newAddress,
      code: poolCode,
      codeHash: ethers.utils.keccak256(poolCode),
    }
  },
  [AccountType.UNISWAP_V3_LIB]: () => {
    return undefined // delete the account
  },
  [AccountType.UNISWAP_V3_OTHER]: async (account, data) => {
    const code = await data.l1MainnetProvider.getCode(account.address)
    return {
      ...account,
      code,
      codeHash: ethers.utils.keccak256(code),
    }
  },
  [AccountType.UNVERIFIED]: () => {
    return undefined // delete the account
  },
  [AccountType.VERIFIED]: () => {
    // TODO
    throw new Error('Not implemented')
  },
}
