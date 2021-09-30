/**
 * Optimism PBC
 */
import path from 'path'

export const SOLC_BIN_PATH = 'https://binaries.soliditylang.org'
export const EMSCRIPTEN_BUILD_PATH = `${SOLC_BIN_PATH}/emscripten-wasm32`
export const EMSCRIPTEN_BUILD_LIST = `${EMSCRIPTEN_BUILD_PATH}/list.json`
export const LOCAL_SOLC_DIR = path.join(__dirname, '..', 'solc-bin')

// Address prefix for predeploy contracts
export const PREDEPLOY = '0x420000000000000000000000000000000000'
// Address prefix for dead contracts
export const DEAD = '0xdeaddeaddeaddeaddeaddeaddeaddeaddead'

export const EOA_CODE_HASHES = [
  '0xa73df79c90ba2496f3440188807022bed5c7e2e826b596d22bcb4e127378835a',
  '0xef2ab076db773ffc554c9f287134123439a5228e92f5b3194a28fec0a0afafe3',
]

export const ECDSA_CONTRACT_ACCOUNT_PREDEPLOY_SLOT =
  '0x0000000000000000000000004200000000000000000000000000000000000003'

export const IMPLEMENTATION_KEY =
  '0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc'

export const skip = [
  '0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24', // ERC 1820 Registery
  '0x06a506a506a506a506a506a506a506a506a506a5', // Gas metadata
]

export const compilerVersionsToSolc = {
  'v0.5.16': 'v0.5.16+commit.9c3226ce',
  'v0.5.16-alpha.7': 'v0.5.16+commit.9c3226ce',
  'v0.6.12': 'v0.6.12+commit.27d51765',
  'v0.7.6': 'v0.7.6+commit.7338295f',
  'v0.7.6+commit.3b061308': 'v0.7.6+commit.7338295f', // what vanilla solidity should this be?
  'v0.7.6-allow_kall': 'v0.7.6+commit.7338295f', // ^same q
  'v0.7.6-no_errors': 'v0.7.6+commit.7338295f',
  'v0.8.4': 'v0.8.4+commit.c7e474f2',
}

export interface EtherscanContract {
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

export interface immutableReference {
  start: number
  length: number
}

export interface immutableReferences {
  [key: string]: immutableReference[]
}
