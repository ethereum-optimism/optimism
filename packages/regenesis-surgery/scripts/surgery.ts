import fs from 'fs'

import { ethers } from 'ethers'
import { add0x, remove0x, clone } from '@eth-optimism/core-utils'

import { StateDump, SurgeryDataSources, AccountType } from './types'
import { findAccount } from './utils'
import { handlers } from './handlers'
import { classify } from './classifiers'
import { loadSurgeryData } from './data'

const doGenesisSurgery = async (
  data: SurgeryDataSources
): Promise<StateDump> => {
  // We'll generate the final genesis file from this output.
  const output: StateDump = []

  // Handle each account in the state dump.
  const input = data.dump.slice(data.configs.startIndex, data.configs.endIndex)

  // Insert any accounts in the genesis that aren't already in the state dump.
  for (const account of data.genesisDump) {
    if (findAccount(input, account.address) === undefined) {
      input.push(account)
    }
  }

  for (const [i, account] of input.entries()) {
    const accountType = classify(account, data)
    console.log(
      `[${i}/${input.length}] ${AccountType[accountType]}: ${account.address}`
    )

    const handler = handlers[accountType]
    const newAccount = await handler(clone(account), data)
    if (newAccount !== undefined) {
      output.push(newAccount)
    }
  }

  // Clean up and standardize the dump. Also performs a few tricks to reduce the overall size of
  // the state dump, which reduces bandwidth requirements.
  console.log('Cleaning up and standardizing dump format...')
  for (const account of output) {
    for (const [key, val] of Object.entries(account)) {
      // We want to be left with the following fields:
      // - balance
      // - nonce
      // - code
      // - storage (if necessary)
      if (key === 'storage') {
        if (Object.keys(account[key]).length === 0) {
          // We don't need storage if there are no storage values.
          delete account[key]
        } else {
          // We can remove 0x from storage keys and vals to save space.
          for (const [storageKey, storageVal] of Object.entries(account[key])) {
            delete account.storage[storageKey]
            account.storage[remove0x(storageKey)] = remove0x(storageVal)
          }
        }
      } else if (key === 'code') {
        // Code MUST start with 0x.
        account[key] = add0x(val)
      } else if (key === 'codeHash' || key === 'root') {
        // Neither of these fields are necessary. Geth will automatically generate them from the
        // code and storage.
        delete account[key]
      } else if (key === 'balance' || key === 'nonce') {
        // At this point we know that the input is either a string or a number. If it's a number,
        // we want to convert it into a string.
        let stripped = typeof val === 'number' ? val.toString(16) : val
        // Remove 0x so we can strip any leading zeros.
        stripped = remove0x(stripped)
        // We can further reduce our genesis size by removing leading zeros. We can even go as far
        // as removing the entire string because Geth appears to treat the empty string as 0.
        stripped = stripped.replace().replace(/^0+/, '')
        // We have to add 0x if the value is greater or equal to than 10 because Geth will throw an
        // error otherwise.
        if (stripped !== '' && ethers.BigNumber.from(add0x(stripped)).gte(10)) {
          stripped = add0x(stripped)
        }
        account[key] = stripped
      } else if (key === 'address') {
        // Keep the address as-is, we'll delete it eventually.
      } else {
        throw new Error(`unexpected account field: ${key}`)
      }
    }
  }

  return output
}

const main = async () => {
  // Load the surgery data.
  const data = await loadSurgeryData()

  // Do the surgery process and get the new genesis dump.
  console.log('Starting surgery process...')
  const finalGenesisDump = await doGenesisSurgery(data)

  // Convert to the format that Geth expects.
  console.log('Converting dump to final format...')
  const finalGenesisAlloc = {}
  for (const account of finalGenesisDump) {
    const address = account.address
    delete account.address
    finalGenesisAlloc[remove0x(address)] = account
  }

  // Attach all of the original genesis configuration values.
  const finalGenesis = {
    ...data.genesis,
    alloc: finalGenesisAlloc,
  }

  // Write the final genesis file to disk.
  console.log('Writing final genesis to disk...')
  fs.writeFileSync(
    data.configs.outputFilePath,
    JSON.stringify(finalGenesis, null, 2)
  )

  console.log('All done!')
}

main()
