import { should } from '../setup'

/* External Imports */
import { bufferUtils, bufToHexString } from '@eth-optimism/core-utils'
import {
  Opcode,
  EVMOpcode,
  EVMBytecode,
  bytecodeToBuffer,
  bufferToBytecode,
  EVMOpcodeAndBytes,
  formatBytecode,
} from '@eth-optimism/rollup-core'

/* Internal imports */
import {
  ErroredTranspilation,
  OpcodeReplacer,
  OpcodeWhitelist,
  SuccessfulTranspilation,
  TranspilationResult,
  Transpiler,
} from '../../src/types/transpiler'
import {
  TranspilerImpl,
  OpcodeReplacerImpl,
  OpcodeWhitelistImpl,
  generateLogSearchTree,
  getJumpIndexSearchBytecode,
} from '../../src/tools/transpiler'
import {
  assertExecutionEqual,
  stateManagerAddress,
  whitelistedOpcodes,
} from '../helpers'
import { EvmIntrospectionUtil } from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'

describe.only('Binary search bytecode generator', () => {
  it('should generate the conceptual tree correctly', () => {
    const keys = [1, 2, 3, 4, 5, 6, 7]
    const values = [8, 9, 10, 11, 12, 13, 14]
    const tree = generateLogSearchTree(keys, values)
    console.log(tree)
  })
  it('should gen some reasonable looking bytecode', () => {
    const keys = [1, 2, 3, 4, 5]
    const vals = [4, 5, 6, 7, 8]
    const bytecode: EVMBytecode = getJumpIndexSearchBytecode(keys, vals, 0)
    console.log(formatBytecode(bytecode))
  })
})
