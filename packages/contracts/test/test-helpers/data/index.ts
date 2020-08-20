import { create2Tests } from './create2.test.json'
import { rlpTests } from './rlp.test.json'
import * as fs from 'fs'
import * as path from 'path'

const createSynthetixJSON = () => {
  const files = {}
  const sanitizeLibs = (str: string): string => {
    return str
      .split('__$')
      .join('000')
      .split('$__')
      .join('000')
  }
  const dir = path.join(__dirname, 'synthetix', 'optimized') + '/'
  fs.readdirSync(dir).forEach((fileName) => {
    if (fileName.endsWith('.json')) {
      const obj = require(dir + fileName)
      files[fileName] = {
        bytecode: '0x' + sanitizeLibs(obj.evm.bytecode.object),
        deployedBytecode: '0x' + sanitizeLibs(obj.evm.deployedBytecode.object),
      }
    }
  })
  return files
}

export interface ContractJSON {
  bytecode: string
  deployedBytecode: string
}
export interface SynthetixBytecode {
  [key: string]: ContractJSON
}
export const SYNTHETIX_BYTECODE: SynthetixBytecode = createSynthetixJSON()

export const CREATE2_TEST_JSON = create2Tests
export const RLP_TEST_JSON = rlpTests
