import { create2Tests } from './create2.test.json'
import { rlpTests } from './rlp.test.json'
import * as fs from 'fs'

const createSynthetixJSON = () => {
  const files = {}
  const sanitizeLibs = (str: string): string => {
    return str
      .split('__$')
      .join('000')
      .split('$__')
      .join('000')
  }
  const dir = __dirname + '/synthetix/unoptimized/'
  console.log(dir)
  fs.readdirSync(dir).forEach((fileName) => {
    if (fileName.endsWith('.json')) {
      const obj = JSON.parse(fs.readFileSync(dir + fileName, 'utf8'))
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
