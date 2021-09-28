import solc from 'solc'
import linker from 'solc/linker'
import { createReadStream, writeFileSync, constants } from 'fs'
import { parseChunked } from '@discoveryjs/json-ext'
import dotenv from 'dotenv'
import fs from 'fs/promises'
import { downloadSolc } from '../src/download-solc'
import { compilerVersionsToSolc, LOCAL_SOLC_DIR } from '../src/constants'

dotenv.config()

const env = process.env
const STATE_DUMP_PATH = env.STATE_DUMP_PATH
const ETHERSCAN_CONTRACTS_PATH = env.ETHERSCAN_CONTRACTS_PATH

interface EtherscanContract {
  contractAddress: string
  code: string
  hash: string
  sourceCode: string
  creationCode: string
  contractFileName: string
  contractName: string
  compilerVersion: string
  optimizationUsed: string
  runs: string
  constructorArguments: string
  library: string
}

;(async () => {
  // First download all required versions of solc
  try {
    await fs.mkdir(LOCAL_SOLC_DIR)
  } catch (e) {
    // directory already exists
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

    if (contract.sourceCode) {
      let input = {
        language: 'Solidity',
        sources: {
          // this is a .sol filename in the example
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
      let sourceCode = contract.sourceCode
      // This is a whole input and Etherscan wraps it around a bracket
      if (sourceCode.substr(0, 2) === '{{') {
        // Trim the first and last bracket
        sourceCode = sourceCode.slice(1, -1)
      }
      try {
        const contractJson = JSON.parse(sourceCode)
        console.log('got json')
        if (contractJson.language) {
          console.log('seems like multifile input')
          input = contractJson
        } else {
          console.error('seems like just the source')
          input.sources = contractJson
        }
      } catch (e) {
        console.error('got error trying json')
      }

      const version = compilerVersionsToSolc[contract.compilerVersion]
      console.log('version', version)
      /* eslint @typescript-eslint/no-var-requires: "off" */
      const currSolc = solc.setupMethods(
        require(`../solc-bin/solc-emscripten-wasm32-${version}.js`)
      )

      const output = JSON.parse(currSolc.compile(JSON.stringify(input)))
      console.log('output', contract.contractAddress, output)
      if (!output.contracts) {
        // There was an error compiling this contract
        noContractsCompiled[contract.contractAddress] = output
        continue
      }

      // Log those without file names
      if (!contract.contractName) {
        noContractName.push(contract.contractAddress)
        continue
      }
      // const mainFile = path.parse(contract.contractFileName).name
      console.log('contractName', contract.contractName)
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

      const immutableRefs = mainOutput.evm.deployedBytecode.immutableReferences
      if (immutableRefs && Object.keys(immutableRefs).length !== 0) {
        console.warn('this contract has immutables', contract.contractAddress)
        hasImmutables[contract.contractAddress] =
          mainOutput.evm.deployedBytecode.immutableReferences
      }
      // Link libraries
      if (contract.library) {
        const deployedBytecode = mainOutput.evm.deployedBytecode.object
        console.log('library', contract.library)
        libraries.push(contract.library)
        const LibToAddress = contract.library.split(':')
        // TEST: just to see output
        console.log(
          'link references!',
          linker.findLinkReferences(deployedBytecode)
        )
        // TODO: empty object should be all the LibToAddressPairs
        // const finalDeployedBytecode = linker.linkBytecode(deployedBytecode, {})
        // use this finalDeployedBytecode to replace in state dump
      }
    }
  }
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

  // TODO: handle immutables
  console.log('has immutables', hasImmutables)
  console.log('all libraries from etherscan file', libraries)

  // TODO: Uniswap: use their published contracts or libraries and just
  // replace with bytecode from there
  // Some contracts will just need to be wiped (those split specifically for OVM)
  // See https://github.com/ethereum-optimism/optimism/pull/1481/files#diff-de41f93baec1842678463433ac56cf5ca6f669d64046729dfbf03dc6b3f03dfeR310-R312
  // for accessing uniswap compiler output
})().catch((err) => {
  console.log(err)
  process.exit(1)
})
