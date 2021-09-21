import { providers, ethers, Contract, BigNumber, utils } from 'ethers'
import solc from 'solc'
import { createReadStream } from 'fs'
import { parseChunked } from '@discoveryjs/json-ext'
import dotenv from 'dotenv'

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
  compilerVersion: string
  optimizationUsed: string
  runs: string
  constructorArguments: string
  library: string
}

const loadRemoteSolc = async (version: string) => {
  return new Promise((resolve, reject) => {
    solc.loadRemoteVersion(version, (err, snapshot) => {
      if (err) {
        console.error('error!', version, err)
      } else {
        resolve(snapshot)
      }
    })
  })
}

;(async () => {
  const etherscanContracts: EtherscanContract[] = await parseChunked(
    createReadStream(ETHERSCAN_CONTRACTS_PATH)
  )

  // Corresponds to vanilla solidity
  const compilerVersions = {
    'v0.5.16': 'v0.5.16+commit.9c3226ce',
    'v0.6.12': 'v0.6.12+commit.27d51765',
    'v0.7.6': 'v0.7.6+commit.7338295f',
    'v0.7.6+commit.3b061308': 'v0.7.6+commit.7338295f', // what vanilla solidity should this be?
    'v0.7.6-allow_kall': 'v0.7.6+commit.7338295f', // ^same q
    'v0.8.4': 'v0.8.4+commit.c7e474f2',
  }

  for (const contract of etherscanContracts) {
    if (contract.sourceCode) {
      console.log(contract)
      const input = {
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
              '*': ['evm.bytecode', 'evm.deployedBytecode', 'abi'],
            },
          },
          optimizer: {
            enabled: contract.optimizationUsed === '1',
            runs: parseInt(contract.runs, 10),
          },
        },
      }

      const version = compilerVersions[contract.compilerVersion]
      console.log('version', version)
      /* eslint @typescript-eslint/no-var-requires: "off" */
      const currSolc = solc.setupMethods(
        require(`../solc-bin/soljson-${version}.js`)
      )
      // Fetching takes a long time
      // const solcSnapshot: any = await loadRemoteSolc(version)
      const output = JSON.parse(currSolc.compile(JSON.stringify(input)))
      console.log('output!', contract.contractAddress, output)

      // How the example called remote
      // solc.loadRemoteVersion(version, (err, solcSnapshot) => {
      //   if (err) {
      //     console.error('error!', version, err) // some have undefined versions!
      //   } else {
      //     const output = JSON.parse(solcSnapshot.compile(JSON.stringify(input)))
      //     console.log('output!', contract.contractAddress, output)
      //   }
      // })
    }
  }
})().catch((err) => {
  console.log(err)
  process.exit(1)
})
