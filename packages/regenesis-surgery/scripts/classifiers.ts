import {
  EOA_CODE_HASHES,
  UNISWAP_V3_FACTORY_ADDRESS,
  UNISWAP_V3_NFPM_ADDRESS,
  UNISWAP_V3_CONTRACT_ADDRESSES,
  UNISWAP_V3_MAINNET_MULTICALL,
  PREDEPLOY_WIPE_ADDRESSES,
  PREDEPLOY_NO_WIPE_ADDRESSES,
  PREDEPLOY_NEW_NOT_ETH_ADDRESSES,
  OLD_ETH_ADDRESS,
  NEW_ETH_ADDRESS,
  ONEINCH_DEPLOYER_ADDRESS,
  DELETE_CONTRACTS,
} from './constants'
import { Account, AccountType, SurgeryDataSources } from './types'
import { hexStringEqual, isBytecodeERC20 } from './utils'

export const classifiers: {
  [key in AccountType]: (account: Account, data: SurgeryDataSources) => boolean
} = {
  [AccountType.ONEINCH_DEPLOYER]: (account) => {
    return hexStringEqual(account.address, ONEINCH_DEPLOYER_ADDRESS)
  },
  [AccountType.DELETE]: (account) => {
    return DELETE_CONTRACTS.some((addr) => {
      return hexStringEqual(account.address, addr)
    })
  },
  [AccountType.EOA]: (account) => {
    // Just in case the account doesn't have a code hash
    if (!account.codeHash) {
      return false
    }

    return EOA_CODE_HASHES.some((codeHash) => {
      return hexStringEqual(account.codeHash, codeHash)
    })
  },
  [AccountType.PRECOMPILE]: (account) => {
    return account.address
      .toLowerCase()
      .startsWith('0x00000000000000000000000000000000000000')
  },
  [AccountType.PREDEPLOY_NEW_NOT_ETH]: (account) => {
    return PREDEPLOY_NEW_NOT_ETH_ADDRESSES.some((addr) => {
      return hexStringEqual(account.address, addr)
    })
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
  [AccountType.UNISWAP_V3_MAINNET_MULTICALL]: (account) => {
    return hexStringEqual(account.address, UNISWAP_V3_MAINNET_MULTICALL)
  },
  [AccountType.UNISWAP_V3_POOL]: (account, data) => {
    return data.pools.some((pool) => {
      return hexStringEqual(pool.oldAddress, account.address)
    })
  },
  [AccountType.UNISWAP_V3_OTHER]: (account) => {
    return UNISWAP_V3_CONTRACT_ADDRESSES.some((addr) => {
      return hexStringEqual(account.address, addr)
    })
  },
  [AccountType.UNVERIFIED]: (account, data) => {
    const found = data.etherscanDump.find(
      (c) => c.contractAddress === account.address
    )
    return found === undefined || found.sourceCode === ''
  },
  [AccountType.VERIFIED]: (account, data) => {
    return !classifiers[AccountType.UNVERIFIED](account, data)
  },
  [AccountType.ERC20]: (account) => {
    return isBytecodeERC20(account.code)
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
