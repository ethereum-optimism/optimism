/**
 * Optimism PBC
 */
import path from 'path'

export const SOLC_BIN_PATH = 'https://binaries.soliditylang.org'
export const EMSCRIPTEN_BUILD_PATH = `${SOLC_BIN_PATH}/emscripten-wasm32`
export const EMSCRIPTEN_BUILD_LIST = `${EMSCRIPTEN_BUILD_PATH}/list.json`
export const LOCAL_SOLC_DIR = path.join(__dirname, '..', 'solc-bin')

export const compilerVersionsToSolc = {
  'v0.5.16': 'v0.5.16+commit.9c3226ce',
  'v0.5.16-alpha.7': 'v0.5.16+commit.9c3226ce',
  'v0.6.12': 'v0.6.12+commit.27d51765',
  'v0.7.6': 'v0.7.6+commit.7338295f',
  'v0.7.6+commit.3b061308': 'v0.7.6+commit.7338295f', // what vanilla solidity should this be?
  'v0.7.6-allow_kall': 'v0.7.6+commit.7338295f', // ^same q
  'v0.8.4': 'v0.8.4+commit.c7e474f2',
}
