import { ethers } from 'ethers'
import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'

interface Account {
  address: string
  nonce: number
  balance: string
  codeHash: string
  root: string
  code?: string
  storage?: {
    [key: string]: string
  }
}

type StateDump = Account[]

enum AccountType {
  EOA,
  PRECOMPILE,
  PREDEPLOY_DEAD,
  PREDEPLOY_WIPE,
  PREDEPLOY_NO_WIPE,
  PREDEPLOY_ETH,
  PREDEPLOY_WETH,
  DEAD,
  UNISWAP_V3_FACTORY,
  UNISWAP_V3_NFPM,
  UNISWAP_V3_POOL,
  UNISWAP_V3_LIB,
  UNISWAP_V3_OTHER,
  UNVERIFIED,
  VERIFIED,
}

interface UniswapPoolData {
  oldAddress: string
  newAddress: string
  token0: string
  token1: string
  fee: ethers.BigNumber
}

interface SurgeryDataSources {
  dump: StateDump
  genesis: StateDump
  pools: UniswapPoolData[]
  l1TestnetProvider: ethers.providers.JsonRpcProvider
  l1MainnetProvider: ethers.providers.JsonRpcProvider
  l2Provider: ethers.providers.JsonRpcProvider
}

/* Constants */

const EOA_CODE_HASHES = [
  '0xa73df79c90ba2496f3440188807022bed5c7e2e826b596d22bcb4e127378835a',
  '0xef2ab076db773ffc554c9f287134123439a5228e92f5b3194a28fec0a0afafe3',
]
const UNISWAP_V3_FACTORY_ADDRESS = null as any // TODO
const UNISWAP_V3_NFPM_ADDRESS = null as any // TODO
const UNISWAP_V3_LIB_ADDRESSES = [
  // Position
  '0x18F7E3ae7202e93984290e1195810c66e1E276FF',
  // Oracle
  '0x17b0f5e5850e7230136df66c5d49497b8c3be0c1',
  // Tick
  '0x47405b0d5f88e16701be6dc8ae185fefaa5dca2f',
  // TickBitmap
  '0x01d95165c3c730d6b40f55c37e24c7aac73d5e6f',
  // TickMath
  '0x308c3e60585ad4eab5b7677be0566fead4cb4746',
  // SwapMath
  '0x198dcc7cd919dd33dd72c3f981df653750901d75',
  // UniswapV3PoolDeployer
  '0x569e8d536ec2dd5988857147c9fcc7d8a08a7dbc',
  // NFTDescriptor
  '0x042f51014b152c2d2fc9b57e36b16bc744065d8c',
]
const UNISWAP_V3_CONTRACT_ADDRESSES = [
  // PoolDeployer
  '0x569E8D536EC2dD5988857147c9FCC7d8a08a7DBc',
  // UniswapV3Factory
  '0x1F98431c8aD98523631AE4a59f267346ea31F984',
  // ProxyAdmin
  '0xB753548F6E010e7e680BA186F9Ca1BdAB2E90cf2',
  // TickLens
  '0xbfd8137f7d1516D3ea5cA83523914859ec47F573',
  // Quoter
  '0xb27308f9F90D607463bb33eA1BeBb41C27CE5AB6',
  // SwapRouter
  '0xE592427A0AEce92De3Edee1F18E0157C05861564',
  // NonfungiblePositionLibrary
  '0x42B24A95702b9986e82d421cC3568932790A48Ec',
  // NonfungibleTokenPositionDescriptor
  '0x91ae842A5Ffd8d12023116943e72A606179294f3',
  // TransparentUpgradeableProxy
  '0xEe6A57eC80ea46401049E92587E52f5Ec1c24785',
  // NonfungibleTokenPositionManager
  '0xC36442b4a4522E871399CD717aBDD847Ab11FE88',
  // UniswapInterfaceMulticall
  '0x1F98415757620B543A52E61c46B32eB19261F984',
]
const PREDEPLOY_WIPE_ADDRESSES = [
  // OVM_GasPriceOracle
  '0x420000000000000000000000000000000000000F',
  // L2StandardBridge
  '0x4200000000000000000000000000000000000010',
  // OVM_SequencerFeeVault
  '0x4200000000000000000000000000000000000011',
  // L2StandardTokenFactory
  '0x4200000000000000000000000000000000000012',
]
const PREDEPLOY_NO_WIPE_ADDRESSES = [
  // OVM_L2ToL1MessagePasser
  '0x4200000000000000000000000000000000000000',
  // OVM_DeployerWhitelist
  '0x4200000000000000000000000000000000000002',
]
const OLD_ETH_ADDRESS = '0x4200000000000000000000000000000000000006'

const hexStringEqual = (a: string, b: string): boolean => {
  if (!ethers.utils.isHexString(a)) {
    throw new Error(`not a hex string: ${a}`)
  }
  if (!ethers.utils.isHexString(b)) {
    throw new Error(`not a hex string: ${b}`)
  }

  return a.toLowerCase() === b.toLowerCase()
}

const isEOAContract = (account: Account): boolean => {
  return EOA_CODE_HASHES.some((codeHash) => {
    return hexStringEqual(account.codeHash, codeHash)
  })
}

const isUniswapV3Factory = (account: Account): boolean => {
  return hexStringEqual(account.address, UNISWAP_V3_FACTORY_ADDRESS)
}

const isUniswapV3NFPM = (account: Account): boolean => {
  return hexStringEqual(account.address, UNISWAP_V3_NFPM_ADDRESS)
}

const isUniswapPool = (account: Account, data: SurgeryDataSources): boolean => {
  return data.pools.some((pool) => {
    return hexStringEqual(pool.oldAddress, account.address)
  })
}

const isUniswapLibrary = (account: Account): boolean => {
  return UNISWAP_V3_LIB_ADDRESSES.some((addr) => {
    return hexStringEqual(account.address, addr)
  })
}

const isUniswapContract = (account: Account): boolean => {
  return UNISWAP_V3_CONTRACT_ADDRESSES.some((addr) => {
    return hexStringEqual(account.address, addr)
  })
}

const isPrecompile = (account: Account): boolean => {
  return account.address
    .toLowerCase()
    .startsWith('0x00000000000000000000000000000000000000')
}

const isPredeployWipe = (account: Account): boolean => {
  return PREDEPLOY_WIPE_ADDRESSES.some((addr) => {
    return hexStringEqual(account.address, addr)
  })
}

const isPredeployNoWipe = (account: Account): boolean => {
  return PREDEPLOY_NO_WIPE_ADDRESSES.some((addr) => {
    return hexStringEqual(account.address, addr)
  })
}

const isPredeployDead = (account: Account): boolean => {
  const PREDEPLOY_DEAD_ADDRESSES = [
    // OVM_L1MessageSender
    '0x4200000000000000000000000000000000000001',
  ]

  return PREDEPLOY_DEAD_ADDRESSES.some((addr) => {
    return hexStringEqual(account.address, addr)
  })
}

const isPredeployETH = (account: Account): boolean => {
  return hexStringEqual(
    account.address,
    '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000'
  )
}

const isPredeployWETH = (account: Account): boolean => {
  return hexStringEqual(account.address, OLD_ETH_ADDRESS)
}

const isEtherscanVerified = (account: Account): boolean => {
  return false // TODO
}

const findAccount = (dump: StateDump, address: string): Account => {
  return dump.find((acc) => {
    return hexStringEqual(acc.address, address)
  })
}

const handlers: {
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
    return null // TODO
  },
  [AccountType.DEAD]: () => {
    return undefined // delete the account
  },
  [AccountType.UNISWAP_V3_FACTORY]: () => {
    // TODO
    // Transfer the owner slot
    // Transfer the feeAmountTickSpacing slot
    // Transfer the getPool slot
    return null // TODO
  },
  [AccountType.UNISWAP_V3_NFPM]: () => {
    // TODO
    // Transfer the _poolIds slot
    return null // TODO
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
    return null // TODO
  },
}

const getAccountType = (
  account: Account,
  data: SurgeryDataSources
): AccountType => {
  if (isEOAContract(account)) {
    return AccountType.EOA
  }

  if (isPrecompile(account)) {
    return AccountType.PRECOMPILE
  }

  if (isPredeployWipe(account)) {
    return AccountType.PREDEPLOY_WIPE
  }

  if (isPredeployNoWipe(account)) {
    return AccountType.PREDEPLOY_NO_WIPE
  }

  if (isPredeployDead(account)) {
    return AccountType.PREDEPLOY_DEAD
  }

  if (isPredeployETH(account)) {
    return AccountType.PREDEPLOY_ETH
  }

  if (isPredeployWETH(account)) {
    return AccountType.PREDEPLOY_WETH
  }

  if (isUniswapV3Factory(account)) {
    return AccountType.UNISWAP_V3_FACTORY
  }

  if (isUniswapV3NFPM(account)) {
    return AccountType.UNISWAP_V3_NFPM
  }

  if (isUniswapPool(account, data)) {
    return AccountType.UNISWAP_V3_POOL
  }

  if (isUniswapLibrary(account)) {
    return AccountType.UNISWAP_V3_LIB
  }

  if (isUniswapContract(account)) {
    return AccountType.UNISWAP_V3_OTHER
  }

  if (isEtherscanVerified(account)) {
    return AccountType.VERIFIED
  } else {
    return AccountType.UNVERIFIED
  }
}

const main = async () => {
  const dump: StateDump = null as any // TODO
  const genesis: StateDump = null as any // TODO
  const pools: UniswapPoolData[] = null as any // TODO
  const data: SurgeryDataSources = {
    dump,
    genesis,
    pools,
    l1TestnetProvider: null as any, // TODO
    l1MainnetProvider: null as any, // TODO
    l2Provider: null as any, // TODO
  }

  // TODO: Insert any accounts from genesis that aren't in the dump

  const output: StateDump = []
  for (const account of dump) {
    const accountType = getAccountType(account, data)
    const handler = handlers[accountType]
    const newAccount = await handler(account, data)
    if (newAccount !== undefined) {
      output.push(newAccount)
    }
  }
}
