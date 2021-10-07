import { ethers } from 'ethers'
import linker from 'solc/linker'
import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'
import {
  POOL_INIT_CODE_HASH_OPTIMISM,
  POOL_INIT_CODE_HASH_OPTIMISM_KOVAN,
} from '@uniswap/v3-sdk'
import { abi as UNISWAP_FACTORY_ABI } from '@uniswap/v3-core/artifacts/contracts/UniswapV3Factory.sol/UniswapV3Factory.json'
import { sleep, add0x, remove0x } from '@eth-optimism/core-utils'
import {
  OLD_ETH_ADDRESS,
  WETH_TRANSFER_ADDRESSES,
  UNISWAP_V3_FACTORY_ADDRESS,
  COMPILER_VERSIONS_TO_SOLC,
} from './constants'
import {
  clone,
  findAccount,
  hexStringIncludes,
  transferStorageSlot,
  getMappingKey,
  getUniswapV3Factory,
  solcInput,
  getSolc,
  getMainContract,
} from './utils'

import {
  Account,
  AccountType,
  SurgeryDataSources,
  ImmutableReference,
  ImmutableReferences,
} from './types'

export const handlers: {
  [key in AccountType]: (
    account: Account,
    data: SurgeryDataSources
  ) => Account | Promise<Account>
} = {
  [AccountType.ONEINCH_DEPLOYER]: (account, data) => {
    return {
      ...handlers[AccountType.EOA](account, data),
      nonce: 0,
    }
  },
  [AccountType.DELETE]: () => {
    return undefined // delete the account
  },
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
    // Get a copy of the old account so we don't modify the one in dump by accident.
    const oldAccount = clone(findAccount(data.dump, OLD_ETH_ADDRESS))

    // Special handling for moving certain balances over to the WETH predeploy.
    // We need to trasnfer all statically defined addresses AND all uni pools.
    const addressesToXfer = WETH_TRANSFER_ADDRESSES.concat(
      data.pools.map((pool) => {
        return pool.oldAddress
      })
    )

    // For each of the listed addresses, check if it has an ETH balance. If so, we remove the ETH
    // balance and give WETH a balance instead.
    let wethBalance = ethers.BigNumber.from(0)
    for (const address of addressesToXfer) {
      const balanceKey = getMappingKey([address], 0)
      if (oldAccount.storage[balanceKey] !== undefined) {
        const accBalance = ethers.BigNumber.from(oldAccount.storage[balanceKey])
        wethBalance = wethBalance.add(accBalance)

        // Remove this balance from the old account storage.
        delete oldAccount.storage[balanceKey]
      }
    }

    const wethBalanceKey = getMappingKey([OLD_ETH_ADDRESS], 0)
    return {
      ...account,
      storage: {
        ...oldAccount.storage,
        ...account.storage,
        [wethBalanceKey]: wethBalance.toHexString(),
      },
    }
  },
  [AccountType.PREDEPLOY_WETH]: async (account, data) => {
    // Treat it like a wipe of the old ETH account.
    account = await handlers[AccountType.PREDEPLOY_WIPE](account, data)

    // Get a copy of the old ETH account so we don't modify the one in dump by accident.
    const ethAccount = clone(findAccount(data.dump, OLD_ETH_ADDRESS))

    // Special handling for moving certain balances over from the old account.
    for (const address of WETH_TRANSFER_ADDRESSES) {
      const balanceKey = getMappingKey([address], 0)
      if (ethAccount.storage[balanceKey] !== undefined) {
        const newBalanceKey = getMappingKey([address], 3)

        // Give this account a balance inside of WETH.
        account.storage[newBalanceKey] = ethAccount.storage[balanceKey]
      }
    }

    // Need to handle pools in a special manner because we want to get the balance for the old pool
    // address but we need to transfer the balance to the new pool address.
    for (const pool of data.pools) {
      const balanceKey = getMappingKey([pool.oldAddress], 0)
      if (ethAccount.storage[balanceKey] !== undefined) {
        const newBalanceKey = getMappingKey([pool.newAddress], 3)

        // Give this account a balance inside of WETH.
        account.storage[newBalanceKey] = ethAccount.storage[balanceKey]
      }
    }

    return account
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
      const UniswapV3Factory = getUniswapV3Factory(data.l1TestnetWallet)
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
  [AccountType.VERIFIED]: (account: Account, data: SurgeryDataSources) => {
    // Final bytecode to be added to the account
    let bytecode: string

    // Make a copy of the account to not mutate it
    account = { ...account }
    // Find the account in the etherscan dump
    const contract = data.etherscanDump.find(
      (c) => c.contractAddress === account.address
    )
    // The contract must exist
    if (!contract) {
      throw new Error(`Unable to find ${account.address} in etherscan dump`)
    }
    // Create the solc input object
    const input = solcInput(contract)
    const version = COMPILER_VERSIONS_TO_SOLC[contract.compilerVersion]
    if (!version) {
      throw new Error(`Unable to find solc version ${contract.compilerVersion}`)
    }

    // Get a solc compiler
    const currSolc = getSolc(version)
    // Compile the contract
    const output = JSON.parse(currSolc.compile(JSON.stringify(input)))
    if (!output.contracts) {
      throw new Error(`Cannot compile ${contract.contractAddress}`)
    }

    // This copies the output so it is safe to mutate below
    const mainOutput = getMainContract(contract, output)
    if (!mainOutput) {
      throw new Error(`Contract filename mismatch: ${contract.contractAddress}`)
    }

    let deployedBytecode = mainOutput.evm.deployedBytecode
    if (typeof deployedBytecode === 'object') {
      deployedBytecode = deployedBytecode.object
    }
    deployedBytecode = add0x(deployedBytecode)

    if (contract.library) {
      console.log('Handling libraries')
      const linkReferences = linker.findLinkReferences(deployedBytecode)

      // The logic only handles linking single libraries. Throw an error in the
      // case where there are multiple libraries.
      if (contract.library.split(':').length > 2) {
        throw new Error(
          `Implement multi library linking handling: ${contract.contractAddress}`
        )
      }

      const [name, address] = contract.library.split(':')
      let key: string
      if (Object.keys(linkReferences).length > 0) {
        key = Object.keys(linkReferences)[0]
      } else {
        key = name
      }

      console.log('Linking')
      deployedBytecode = linker.linkBytecode(deployedBytecode, {
        [key]: add0x(address),
      })
    }

    bytecode = add0x(deployedBytecode)

    // If the contract has immutables in it, then the contracts
    // need to be compiled with the ovm compiler so that the offsets
    // can be found. The immutables must be pulled out of the old code
    // and inserted into the new code
    const immutableRefs: ImmutableReference =
      mainOutput.evm.deployedBytecode.immutableReferences
    if (immutableRefs && Object.keys(immutableRefs).length !== 0) {
      console.log('Handling immutables')
      // Compile using the ovm compiler to find the location of the
      // immutableRefs in the ovm contract so they can be migrated
      // to the new contract
      const ovmSolc = getSolc(contract.compilerVersion, true)
      const ovmOutput = JSON.parse(ovmSolc.compile(JSON.stringify(input)))
      const ovmFile = getMainContract(contract, ovmOutput)
      if (!ovmFile) {
        throw new Error(
          `Contract filename mismatch: ${contract.contractAddress}`
        )
      }

      const ovmImmutableRefs: ImmutableReference =
        ovmFile.evm.deployedBytecode.immutableReferences

      let ovmObject = ovmFile.evm.deployedBytecode
      if (typeof ovmObject === 'object') {
        ovmObject = ovmObject.object
      }

      ovmObject = add0x(ovmObject)

      // Iterate over the immutableRefs and slice them into the new code
      // to carry over their values. The keys are the AST IDs
      for (const [key, value] of Object.entries(immutableRefs)) {
        const ovmValue = ovmImmutableRefs[key]
        if (!ovmValue) {
          throw new Error(`cannot find ast in ovm compiler output`)
        }
        // Each value is an array of {length, start}
        for (const [i, ref] of value.entries()) {
          const ovmRef = ovmValue[i]
          if (ref.length !== ovmRef.length) {
            throw new Error(`length mismatch`)
          }

          // Get the value from the contract code
          const immutable = ethers.utils.hexDataSlice(
            add0x(account.code),
            ovmRef.start,
            ovmRef.start + ovmRef.length
          )

          deployedBytecode = add0x(deployedBytecode)

          const pre = ethers.utils.hexDataSlice(deployedBytecode, 0, ref.start)
          const post = ethers.utils.hexDataSlice(
            deployedBytecode,
            ref.start + ref.length
          )
          // Assign to the global bytecode variable
          bytecode = ethers.utils.hexConcat([pre, immutable, post])

          if (bytecode.length !== deployedBytecode.length) {
            throw new Error(
              `mismatch in size: ${bytecode.length} vs ${deployedBytecode.length}`
            )
          }
        }
      }
    }

    // Handle migrating storage slots
    if (account.storage) {
      console.log('Handling storage')
      for (const pool of data.pools) {
        // Check for references to modified values in storage.
        for (const [key, value] of Object.entries(account.storage)) {
          // Turn into hex string or hexStringIncludes will throw
          const val = add0x(value)
          if (hexStringIncludes(val, pool.oldAddress)) {
            console.log(
              `found unexpected reference to pool address ${val} in ${account.address}`
            )
            const regex = new RegExp(
              remove0x(pool.oldAddress).toLowerCase(),
              'g'
            )
            account.storage[key] = value.replace(
              regex,
              remove0x(pool.newAddress).toLowerCase()
            )
            console.log(`updated to ${account.storage[key]}`)
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

        if (data.l2NetworkName === 'mainnet') {
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
    }

    account.code = remove0x(bytecode)
    account.codeHash = ethers.utils.keccak256(add0x(bytecode))

    return account
  },
}
