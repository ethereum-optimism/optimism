import {
  EOA_CODE_HASHES,
  UNISWAP_V3_FACTORY_ADDRESS,
  UNISWAP_V3_NFPM_ADDRESS,
  UNISWAP_V3_LIB_ADDRESSES,
  UNISWAP_V3_CONTRACT_ADDRESSES,
  PREDEPLOY_WIPE_ADDRESSES,
  PREDEPLOY_NO_WIPE_ADDRESSES,
  PREDEPLOY_DEAD_ADDRESSES,
  OLD_ETH_ADDRESS,
  NEW_ETH_ADDRESS,
} from './constants'
import { Account, AccountType, SurgeryDataSources } from './types'
import { hexStringEqual } from './utils'

export const classifiers: {
  [key in AccountType]: (account: Account, data: SurgeryDataSources) => boolean
} = {
  [AccountType.EOA]: (account) => {
    return EOA_CODE_HASHES.some((codeHash) => {
      return hexStringEqual(account.codeHash, codeHash)
    })
  },
  [AccountType.PRECOMPILE]: (account) => {
    return account.address
      .toLowerCase()
      .startsWith('0x00000000000000000000000000000000000000')
  },
  [AccountType.PREDEPLOY_WIPE]: (account) => {
    return PREDEPLOY_WIPE_ADDRESSES.some((addr) => {
      return hexStringEqual(account.address, addr)
    })
  },
  [AccountType.PREDEPLOY_NO_WIPE]: (account) => {
    return PREDEPLOY_NO_WIPE_ADDRESSES.some((addr) => {
      return hexStringEqual(account.address, addr)
    })
  },
  [AccountType.PREDEPLOY_DEAD]: (account) => {
    return PREDEPLOY_DEAD_ADDRESSES.some((addr) => {
      return hexStringEqual(account.address, addr)
    })
  },
  [AccountType.PREDEPLOY_ETH]: (account) => {
    return hexStringEqual(account.address, NEW_ETH_ADDRESS)
  },
  [AccountType.PREDEPLOY_WETH]: (account) => {
    return hexStringEqual(account.address, OLD_ETH_ADDRESS)
  },
  [AccountType.UNISWAP_V3_FACTORY]: (account) => {
    return hexStringEqual(account.address, UNISWAP_V3_FACTORY_ADDRESS)
  },
  [AccountType.UNISWAP_V3_NFPM]: (account) => {
    return hexStringEqual(account.address, UNISWAP_V3_NFPM_ADDRESS)
  },
  [AccountType.UNISWAP_V3_POOL]: (account, data) => {
    return data.pools.some((pool) => {
      return hexStringEqual(pool.oldAddress, account.address)
    })
  },
  [AccountType.UNISWAP_V3_LIB]: (account) => {
    return UNISWAP_V3_LIB_ADDRESSES.some((addr) => {
      return hexStringEqual(account.address, addr)
    })
  },
  [AccountType.UNISWAP_V3_OTHER]: (account) => {
    return UNISWAP_V3_CONTRACT_ADDRESSES.some((addr) => {
      return hexStringEqual(account.address, addr)
    })
  },
  [AccountType.VERIFIED]: (account) => {
    // TODO
    throw new Error('Not implemented')
  },
  [AccountType.UNVERIFIED]: (account) => {
    // TODO
    throw new Error('Not implemented')
  },
}

export const classify = (
  account: Account,
  data: SurgeryDataSources
): AccountType => {
  for (const accountType in AccountType) {
    if (!isNaN(Number(accountType))) {
      if (classifiers[accountType](account, data)) {
        return Number(accountType)
      }
    }
  }
}
