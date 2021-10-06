import { ethers } from 'ethers'
import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'
import {
  POOL_INIT_CODE_HASH_OPTIMISM,
  POOL_INIT_CODE_HASH_OPTIMISM_KOVAN,
} from '@uniswap/v3-sdk'
import { abi as UNISWAP_FACTORY_ABI } from '@uniswap/v3-core/artifacts/contracts/UniswapV3Factory.sol/UniswapV3Factory.json'
import { sleep } from '@eth-optimism/core-utils'
import { OLD_ETH_ADDRESS, UNISWAP_V3_FACTORY_ADDRESS } from './constants'
import { Account, AccountType, SurgeryDataSources } from './types'
import {
  findAccount,
  hexStringIncludes,
  transferStorageSlot,
  getMappingKey,
} from './utils'

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
  [AccountType.UNISWAP_V3_FACTORY]: async (account, data) => {
    // Transfer the owner slot
    transferStorageSlot({
      account,
      oldSlot: 0,
      newSlot: 3,
    })

    // Transfer the feeAmountTickSpacing slot
    for (const fee of [500, 3000, 10000]) {
      transferStorageSlot({
        account,
        oldSlot: getMappingKey([fee], 1),
        newSlot: getMappingKey([fee], 4),
      })
    }

    // Transfer the getPool slot
    for (const pool of data.pools) {
      // Fix the token0 => token1 => fee mapping
      transferStorageSlot({
        account,
        oldSlot: getMappingKey([pool.token0, pool.token1, pool.fee], 2),
        newSlot: getMappingKey([pool.token0, pool.token1, pool.fee], 5),
        newValue: pool.newAddress,
      })

      // Fix the token1 => token0 => fee mapping
      transferStorageSlot({
        account,
        oldSlot: getMappingKey([pool.token1, pool.token0, pool.fee], 2),
        newSlot: getMappingKey([pool.token1, pool.token0, pool.fee], 5),
        newValue: pool.newAddress,
      })
    }

    return handlers[AccountType.UNISWAP_V3_OTHER](account, data)
  },
  [AccountType.UNISWAP_V3_NFPM]: async (account, data) => {
    for (const pool of data.pools) {
      try {
        transferStorageSlot({
          account,
          oldSlot: getMappingKey([pool.oldAddress], 10),
          newSlot: getMappingKey([pool.newAddress], 10),
        })
      } catch (err) {
        if (err.message.includes('old slot not found in state dump')) {
          // It's OK for this to happen because some pools may not have any position NFTs.
          console.log(
            `pool not found in NonfungiblePositionManager _poolIds mapping: ${pool.oldAddress}`
          )
        } else {
          throw err
        }
      }
    }

    return handlers[AccountType.UNISWAP_V3_OTHER](account, data)
  },
  [AccountType.UNISWAP_V3_POOL]: async (account, data) => {
    // Find the pool by its old address
    const pool = data.pools.find((poolData) => {
      return poolData.oldAddress === account.address
    })

    // Get the pool's code.
    let poolCode = await data.l1TestnetProvider.getCode(pool.newAddress)
    if (poolCode === '0x') {
      const UniswapV3Factory = new ethers.Contract(
        UNISWAP_V3_FACTORY_ADDRESS,
        UNISWAP_FACTORY_ABI,
        data.l1TestnetWallet
      )

      await UniswapV3Factory.createPool(pool.token0, pool.token1, pool.fee)

      let retries = 0
      while (poolCode === '0x') {
        retries++
        if (retries > 50) {
          throw new Error(`unable to create pool with data: ${pool}`)
        }

        poolCode = await data.l1TestnetProvider.getCode(pool.newAddress)
        await sleep(5000)
      }
    }

    return {
      ...account,
      address: pool.newAddress,
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
  [AccountType.VERIFIED]: (account, data) => {
    if (account.storage) {
      for (const pool of data.pools) {
        // Check for references to modified values in storage.
        for (const [key, val] of Object.entries(account.storage)) {
          // TODO: Do we need to do anything if these statements trigger?
          if (hexStringIncludes(val, pool.oldAddress)) {
            throw new Error(`found unexpected reference to pool address`)
          }

          if (hexStringIncludes(val, POOL_INIT_CODE_HASH_OPTIMISM)) {
            throw new Error(
              `found unexpected reference to mainnet pool init code hash`
            )
          }

          if (hexStringIncludes(val, POOL_INIT_CODE_HASH_OPTIMISM_KOVAN)) {
            throw new Error(
              `found unexpected reference to kovan pool init code hash`
            )
          }
        }

        // Fix single-level mappings (e.g., balance mappings)
        for (let i = 0; i < 1000; i++) {
          const oldSlotKey = getMappingKey([pool.oldAddress], i)
          if (account.storage[oldSlotKey] !== undefined) {
            console.log(
              `fixing single-level mapping in contract`,
              `address=${account.address}`,
              `pool=${pool.oldAddress}`,
              `slot=${oldSlotKey}`
            )
            transferStorageSlot({
              account,
              oldSlot: oldSlotKey,
              newSlot: getMappingKey([pool.newAddress], i),
            })
          }
        }

        // Fix double-level mappings (e.g., allowance mappings)
        for (let i = 0; i < 1000; i++) {
          for (const otherAccount of data.dump) {
            // otherAddress => poolAddress => xxxx
            const oldSlotKey1 = getMappingKey(
              [otherAccount.address, pool.oldAddress],
              i
            )
            if (account.storage[oldSlotKey1] !== undefined) {
              console.log(
                `fixing double-level mapping in contract (other => pool => xxxx)`,
                `address=${account.address}`,
                `pool=${pool.oldAddress}`,
                `slot=${oldSlotKey1}`
              )
              transferStorageSlot({
                account,
                oldSlot: oldSlotKey1,
                newSlot: getMappingKey(
                  [otherAccount.address, pool.newAddress],
                  i
                ),
              })
            }

            // poolAddress => otherAddress => xxxx
            const oldSlotKey2 = getMappingKey(
              [pool.oldAddress, otherAccount.address],
              i
            )
            if (account.storage[oldSlotKey2] !== undefined) {
              console.log(
                `fixing double-level mapping in contract (pool => other => xxxx)`,
                `address=${account.address}`,
                `pool=${pool.oldAddress}`,
                `slot=${oldSlotKey2}`
              )
              transferStorageSlot({
                account,
                oldSlot: oldSlotKey2,
                newSlot: getMappingKey(
                  [pool.newAddress, otherAccount.address],
                  i
                ),
              })
            }
          }
        }
      }
    }

    // TODO
    throw new Error('Not implemented')
  },
}
