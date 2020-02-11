/* External Imports */
import { bufToHexString } from '@eth-optimism/core-utils'
import {
  bytecodeToBuffer,
  EVMBytecode,
  Opcode,
} from '@eth-optimism/rollup-core'
import * as fs from 'fs'
import { resolve } from 'path'

// describe('Used to generate transpilation input files', () => {
//   it('Generates a file to test/transpiler/util/generated/input.bytecode', () => {
//     const bytecode: EVMBytecode = [
//       { opcode: Opcode.PUSH1, consumedBytes: undefined },
//       { opcode: Opcode.SLOAD, consumedBytes: undefined },
//       { opcode: Opcode.SLOAD, consumedBytes: undefined },
//       { opcode: Opcode.PUSH1, consumedBytes: undefined },
//     ]
//
//     const bytes: Buffer = bytecodeToBuffer(bytecode)
//     console.log(`Hex Bytes: ${bufToHexString(bytes)}`)
//
//     const generatedDir: string = resolve(
//       __dirname,
//       '../../../test/transpiler/util/generated'
//     )
//
//     if (!fs.existsSync(generatedDir)) {
//       fs.mkdirSync(generatedDir)
//     }
//
//     fs.writeFileSync(`${generatedDir}/input.bytecode`, bytes)
//   })
// })
