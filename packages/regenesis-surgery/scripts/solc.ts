/* eslint @typescript-eslint/no-var-requires: "off" */
import { access, mkdir } from 'fs/promises'
import fetch from 'node-fetch'
import path from 'path'
import fs from 'fs'
import solc from 'solc'
import { ethers } from 'ethers'
import { clone } from '@eth-optimism/core-utils'
import {
  COMPILER_VERSIONS_TO_SOLC,
  EMSCRIPTEN_BUILD_LIST,
  EMSCRIPTEN_BUILD_PATH,
  LOCAL_SOLC_DIR,
} from './constants'
import { EtherscanContract } from './types'

const OVM_BUILD_PATH = (version: string) => {
  return `https://raw.githubusercontent.com/ethereum-optimism/solc-bin/9455107699d2f7ad9b09e1005c7c07f4b5dd6857/bin/soljson-${version}.js`
}

/**
 * Downloads a specific solc version.
 *
 * @param version Solc version to download.
 * @param ovm If true, downloads from the OVM repository.
 */
export const downloadSolc = async (version: string, ovm?: boolean) => {
  // TODO: why is this one missing?
  if (version === 'v0.5.16-alpha.7') {
    return
  }

  // File is the location where we'll put the downloaded compiler.
  let file: string
  // Remote is the URL we'll query if the file doesn't already exist.
  let remote: string

  // Exact file/remote will depend on if downloading OVM or EVM compiler.
  if (ovm) {
    file = `${path.join(LOCAL_SOLC_DIR, version)}.js`
    remote = OVM_BUILD_PATH(version)
  } else {
    const res = await fetch(EMSCRIPTEN_BUILD_LIST)
    const data: any = await res.json()
    const list = data.builds

    // Make sure the target version actually exists
    let target: any
    for (const entry of list) {
      const longVersion = `v${entry.longVersion}`
      if (version === longVersion) {
        target = entry
      }
    }

    // Error out if the given version can't be found
    if (!target) {
      throw new Error(`Cannot find compiler version ${version}`)
    }

    file = path.join(LOCAL_SOLC_DIR, target.path)
    remote = `${EMSCRIPTEN_BUILD_PATH}/${target.path}`
  }

  try {
    // Check to see if we already have the file
    await access(file, fs.constants.F_OK)
  } catch (e) {
    console.error(`Downloading ${version} ${ovm ? 'ovm' : 'solidity'}`)
    // If we don't have the file, download it
    const res = await fetch(remote)
    const bin = await res.text()
    fs.writeFileSync(file, bin)
  }
}

/**
 * Downloads all required solc versions, if not already downloaded.
 */
export const downloadAllSolcVersions = async () => {
  try {
    await mkdir(LOCAL_SOLC_DIR)
  } catch (e) {
    // directory already exists
  }

  // Keys are OVM versions.
  await Promise.all(
    // Use a set to dedupe the list of versions.
    [...new Set(Object.keys(COMPILER_VERSIONS_TO_SOLC))].map(
      async (version) => {
        await downloadSolc(version, true)
      }
    )
  )

  // Values are EVM versions.
  await Promise.all(
    // Use a set to dedupe the list of versions.
    [...new Set(Object.values(COMPILER_VERSIONS_TO_SOLC))].map(
      async (version) => {
        await downloadSolc(version)
      }
    )
  )
}

export const getMainContract = (contract: EtherscanContract, output) => {
  if (contract.contractFileName) {
    return clone(
      output.contracts[contract.contractFileName][contract.contractName]
    )
  }
  return clone(output.contracts.file[contract.contractName])
}

export const getSolc = (version: string, ovm?: boolean) => {
  return solc.setupMethods(
    require(path.join(
      LOCAL_SOLC_DIR,
      ovm ? version : `solc-emscripten-wasm32-${version}.js`
    ))
  )
}

export const solcInput = (contract: EtherscanContract) => {
  // Create a base solc input object
  const input = {
    language: 'Solidity',
    sources: {
      file: {
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
    // source code may be one of 3 things
    // - raw content string
    // - sources object
    // - entire input
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
    // If the json has language, then it is the whole input
    if (json.language) {
      return json
    }
    // Add the json file as the sources
    input.sources = json
  } catch (e) {
    //
  }
  return input
}

const compilerCache: {
  [hash: string]: any
} = {}

export const compile = (opts: {
  contract: EtherscanContract
  ovm: boolean
}): any => {
  let version: string
  if (opts.ovm) {
    version = opts.contract.compilerVersion
  } else {
    version = COMPILER_VERSIONS_TO_SOLC[opts.contract.compilerVersion]
    if (!version) {
      throw new Error(
        `Unable to find solc version ${opts.contract.compilerVersion}`
      )
    }
  }

  const solcInstance = getSolc(version, opts.ovm)
  const input = JSON.stringify(solcInput(opts.contract))
  const inputHash = ethers.utils.solidityKeccak256(['string'], [input])

  // Cache the compiler output to speed up repeated compilations of the same contract. If this
  // cache is too memory intensive, then we could consider only caching if the contract has been
  // seen more than once.
  let output: any
  if (compilerCache[inputHash]) {
    output = compilerCache[inputHash]
  } else {
    output = JSON.parse(solcInstance.compile(input))
    compilerCache[inputHash] = output
  }

  if (!output.contracts) {
    throw new Error(`Cannot compile ${opts.contract.contractAddress}`)
  }

  const mainOutput = getMainContract(opts.contract, output)
  if (!mainOutput) {
    throw new Error(
      `Contract filename mismatch: ${opts.contract.contractAddress}`
    )
  }

  return mainOutput
}
