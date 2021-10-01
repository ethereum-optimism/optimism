/**
 * Optimism PBC
 */

/* eslint @typescript-eslint/no-var-requires: "off" */

import solc from 'solc'
import linker from 'solc/linker'
import { createReadStream, writeFileSync, constants } from 'fs'
import { parseChunked } from '@discoveryjs/json-ext'
import dotenv from 'dotenv'
import fs from 'fs/promises'
import { downloadSolc } from '../src/download-solc'
import {
  compilerVersionsToSolc,
  LOCAL_SOLC_DIR,
  EtherscanContract,
  EOA_CODE_HASHES,
  ECDSA_CONTRACT_ACCOUNT_PREDEPLOY_SLOT,
  IMPLEMENTATION_KEY,
  skip,
  immutableReference,
  immutableReferences,
} from '../src/constants'
import {
  isPredeploy,
  isDeadAddress,
  hasSourceCode,
  isSafeToSkip,
  isEOA,
  solcInput,
} from '../src/helpers'
import { ethers } from 'ethers'

dotenv.config()

const env = process.env
const STATE_DUMP_PATH = env.STATE_DUMP_PATH
const ETHERSCAN_CONTRACTS_PATH = env.ETHERSCAN_CONTRACTS_PATH
const ETHEREUM_HTTP_URL = env.ETHEREUM_HTTP_URL

const provider = new ethers.providers.JsonRpcProvider(ETHEREUM_HTTP_URL)

;(async () => {
  // First download all required versions of solc
  try {
    await fs.mkdir(LOCAL_SOLC_DIR)
  } catch (e) {
    // directory already exists
  }

  for (const version of Object.keys(compilerVersionsToSolc)) {
    await downloadSolc(version, true) // using ovm
  }
  for (const version of Object.values(compilerVersionsToSolc)) {
    await downloadSolc(version)
  }

  // Read state dump from disk
  const etherscanContracts: EtherscanContract[] = await parseChunked(
    createReadStream(ETHERSCAN_CONTRACTS_PATH)
  )

  const noContractsCompiled = {}
  const noContractName = []
  const contractFileNameMismatch = {}
  const hasImmutables = {}
  const libraries = []

  // Iterate through the contracts
  for (const contract of etherscanContracts) {
    // TODO: sanity check the contract before processing it
    // require certain fields exist

    // Skip processing of predeploy contracts
    if (isPredeploy(contract)) {
      console.error(`Skipping predeploy ${contract.contractAddress}`)
      continue
    }
    // Skip processing of system contracts
    if (isDeadAddress(contract)) {
      console.error(`Skipping dead address ${contract.contractAddress}`)
      continue
    }

    // Some contracts are safe to skip. Each contract that is
    // safe to skip must be inspected manually
    if (isSafeToSkip(contract)) {
      continue
    }

    // Skip processing of EOAs and warn for other unknown contracts.
    // These should be recorded and followed up with manually
    if (!hasSourceCode(contract)) {
      const eoa = await isEOA(contract, provider)
      if (!eoa) {
        console.error(`unknown contract ${contract.contractAddress}`)
      }
      continue
    }

    // Process contracts that have source code
    if (hasSourceCode(contract)) {
      console.error(
        `Found contract with source code: ${contract.contractAddress}`
      )
      const input = solcInput(contract)
      const version = compilerVersionsToSolc[contract.compilerVersion]
      if (!version) {
        throw new Error(
          `Unable to find solc version ${contract.compilerVersion}`
        )
      }

      // TODO: turn this path into a constant or add a helper function
      // that returns a solc-js instance
      const currSolc = solc.setupMethods(
        require(`../solc-bin/solc-emscripten-wasm32-${version}.js`)
      )

      // Compile the contract
      const output = JSON.parse(currSolc.compile(JSON.stringify(input)))
      if (!output.contracts) {
        console.error(`Cannot compile ${contract.contractAddress}`)
        // There was an error compiling this contract
        noContractsCompiled[contract.contractAddress] = output
        continue
      }

      // Log those without file names
      if (!contract.contractName) {
        console.error(`Found contract without name ${contract.contractAddress}`)
        noContractName.push(contract.contractAddress)
        continue
      }

      // TODO: How can we make sure this is correct?
      // Contract name does not correspond with what's compiled from Etherscan sourcecode
      let mainOutput
      // there's a name for this multi-file address
      if (contract.contractFileName) {
        mainOutput =
          output.contracts[contract.contractFileName][contract.contractName]
      } else {
        mainOutput = output.contracts.file[contract.contractName]
      }
      if (!mainOutput) {
        contractFileNameMismatch[contract.contractAddress] = contract
        continue
      }

      // Find the immutables in the old code and move them to the new
      const immutableRefs: immutableReference =
        mainOutput.evm.deployedBytecode.immutableReferences
      if (immutableRefs && Object.keys(immutableRefs).length !== 0) {
        // Compile using the ovm compiler to find the location of the
        // immutableRefs in the ovm contract so they can be migrated
        // to the new contract
        const ovmSolc = solc.setupMethods(
          require(`../solc-bin/${contract.compilerVersion}.js`)
        )
        const ovmOutput = JSON.parse(ovmSolc.compile(JSON.stringify(input)))
        let ovmFile
        if (contract.contractFileName) {
          ovmFile =
            ovmOutput.contracts[contract.contractFileName][contract.contractName]
        } else {
          ovmFile = ovmOutput.contracts.file[contract.contractName]
        }

        const ovmImmutableRefs: immutableReference =
          ovmFile.evm.deployedBytecode.immutableReferences

        let ovmObject = ovmFile.evm.deployedBytecode
        if (typeof ovmObject === 'object') {
          ovmObject = ovmObject.object
        }

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
            const immutable = ovmObject.slice(
              ovmRef.start,
              ovmRef.start + ovmRef.length
            )
            console.error(`Found immutable: ${immutable}`)

            let object = mainOutput.evm.deployedBytecode
            if (object === undefined) {
              throw new Error(`deployedBytecode undefined`)
            }
            // Sometimes the shape of the output is different?
            if (typeof object === 'object') {
              object = object.object
            }

            const pre = object.slice(0, ref.start)
            const post = object.slice(ref.start + ref.length)
            const bytecode = pre + immutable + post

            if (bytecode.length !== object.length) {
              throw new Error(
                `mismatch in size: ${bytecode.length} vs ${object.length}`
              )
            }

            // TODO: double check this is correct
            mainOutput.evm.deployedBytecode = bytecode
          }
        }

        console.warn('this contract has immutables', contract.contractAddress)
        hasImmutables[contract.contractAddress] =
          mainOutput.evm.deployedBytecode.immutableReferences
      }

      // Link libraries
      if (contract.library) {
        const deployedBytecode = mainOutput.evm.deployedBytecode.object
        //console.log('library', contract.library)
        libraries.push(contract.library)
        const LibToAddress = contract.library.split(':')
        // TEST: just to see output
        /*
        console.log(
          'link references!',
          linker.findLinkReferences(deployedBytecode)
        )
        */
        // TODO: empty object should be all the LibToAddressPairs
        // const finalDeployedBytecode = linker.linkBytecode(deployedBytecode, {})
        // use this finalDeployedBytecode to replace in state dump
      }
    }
  }

  console.log('all done')
  /*
  console.log('had compiler errors', noContractsCompiled)
  for (const [address, output] of Object.entries(noContractsCompiled)) {
    console.log('error at address', address)
    console.log(output)
  }
  console.log('filename missing from etherscan file', noContractName)
  console.log(
    'filename not found in compiled contracts',
    contractFileNameMismatch
  )
  */

  // TODO: handle immutables
  //console.log('has immutables', hasImmutables)
  //console.log('all libraries from etherscan file', libraries)

  // TODO: Uniswap: use their published contracts or libraries and just
  // replace with bytecode from there
  // Some contracts will just need to be wiped (those split specifically for OVM)
  // See https://github.com/ethereum-optimism/optimism/pull/1481/files#diff-de41f93baec1842678463433ac56cf5ca6f669d64046729dfbf03dc6b3f03dfeR310-R312
  // for accessing uniswap compiler output
})().catch((err) => {
  console.log(err)
  process.exit(1)
})
