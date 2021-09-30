/**
 * Optimism PBC
 */

import { ethers } from 'ethers'
import { BaseProvider } from '@ethersproject/providers'

import {
  PREDEPLOY,
  DEAD,
  compilerVersionsToSolc,
  LOCAL_SOLC_DIR,
  EtherscanContract,
  EOA_CODE_HASHES,
  ECDSA_CONTRACT_ACCOUNT_PREDEPLOY_SLOT,
  IMPLEMENTATION_KEY,
  skip,
} from '../src/constants'

export const isPredeploy = (contract: EtherscanContract): boolean => {
  return contract.contractAddress.startsWith(PREDEPLOY)
}

export const isDeadAddress = (contract: EtherscanContract): boolean => {
  return contract.contractAddress.startsWith(DEAD)
}

export const hasSourceCode = (contract: EtherscanContract): boolean => {
  return contract.sourceCode !== ''
}

export const isSafeToSkip = (contract: EtherscanContract): boolean => {
  return skip.includes(contract.contractAddress)
}

export const isEOA = async (
  contract: EtherscanContract,
  provider: BaseProvider
): Promise<boolean> => {
  // EOAs had smart contract wallets
  if (contract.code === '0x') {
    return false
  }
  const codeHash = ethers.utils.keccak256(contract.code)
  if (EOA_CODE_HASHES.includes(codeHash)) {
    return true
  }
  const slot = await provider.getStorageAt(
    contract.contractAddress,
    IMPLEMENTATION_KEY
  )
  if (slot === ECDSA_CONTRACT_ACCOUNT_PREDEPLOY_SLOT) {
    return true
  }
  return false
}

export const solcInput = (contract: EtherscanContract) => {
  // Create a base solc input object
  const input = {
    language: 'Solidity',
    sources: {
      file: {
        // TODO: does this need the brackets?
        content: contract.sourceCode,
      },
    },
    settings: {
      outputSelection: {
        '*': {
          '*': ['*'],
        },
      },
      optimizer: {
        enabled: contract.optimizationUsed === '1',
        runs: parseInt(contract.runs, 10),
      },
    },
  }

  try {
    let sourceCode = contract.sourceCode
    // Remove brackets that are wrapped around the source
    // when trying to parse json
    if (sourceCode.substr(0, 2) === '{{') {
      // Trim the first and last bracket
      sourceCode = sourceCode.slice(1, -1)
    }
    // If the source code is valid json, and
    // has the keys of a solc input, just return it
    const json = JSON.parse(sourceCode)
    if (json.language) {
      return json
    }
    // Add the json file as the sources
    input.sources = json
  } catch (e) {
    console.error(`Unable to parse json ${contract.contractAddress}`)
  }
  return input
}
