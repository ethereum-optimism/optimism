import { should } from '../setup'

/* External Imports */
import { getLogger, Logger, bufferUtils, bufToHexString, hexStrToBuf, BigNumber } from '@pigi/core-utils'
import {
  Opcode,
  EVMOpcode,
  EVMBytecode,
  bytecodeToBuffer,
  bufferToBytecode,
  EVMOpcodeAndBytes,
  formatBytecode,
} from '@pigi/rollup-core'

/* Internal imports */
import {
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
  dynamicStashMemoryInStack,
  dynamicUnstashMemoryFromStack,
  staticStashMemoryInStack,
  staticUnstashMemoryFromStack,
  getPUSHIntegerOp
} from '../../src/tools/transpiler'
import {
  assertExecutionEqual,
  stateManagerAddress,
  whitelistedOpcodes,
} from './helpers'
import { EvmIntrospectionUtil } from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import { exec } from 'child_process'

const log: Logger = getLogger('test-memory-sub')

const storeNWordsInMemorySequential = function (numWords: number): EVMBytecode {
    let storageBytecode: EVMBytecode = []
    for (let i = 0; i < numWords; i++) {
        storageBytecode = storageBytecode.concat([
            {
                opcode: Opcode.PUSH32,
                consumedBytes: Buffer.alloc(32).fill(new BigNumber(i).toBuffer('B', 1))
            },
            {
                opcode: Opcode.PUSH32,
                consumedBytes: new BigNumber((i) * 32).toBuffer('B', 32)
            },
            {
                opcode: Opcode.MSTORE,
                consumedBytes: undefined
            }
        ])
    }
    return storageBytecode
}

const overwriteNWordsInMemoryWithOffset = function (numWords: number, offset: number): EVMBytecode {
    let overwriteBytecode: EVMBytecode = []
    for (let i = 0; i < numWords; i++) {
        overwriteBytecode = overwriteBytecode.concat([
            {
                opcode: Opcode.PUSH32,
                consumedBytes: hexStrToBuf('0x6969696969696969696969696969696969696969696969696969696969696969') // nice
            },
            getPUSHIntegerOp(offset + i * 32),
            {
                opcode: Opcode.MSTORE,
                consumedBytes: undefined
            }
        ])
    }
    return overwriteBytecode
}


describe.only('Memory Replacement Operations', () => {
  let opcodeWhitelist: OpcodeWhitelist
  let transpiler: Transpiler
  let replacer: OpcodeReplacer
  let evmUtil: EvmIntrospectionUtil
  let storeSomeValsInMemory: EVMBytecode = []


  before(() => {
    storeSomeValsInMemory = storeNWordsInMemorySequential(3)
    // insert a random opcode to replace inside
    const pointlessOperation: EVMBytecode = [
        {
            opcode: Opcode.PUSH1,
            consumedBytes: hexStrToBuf('0xaa')
        },
        {
            opcode: Opcode.POP,
            consumedBytes: undefined
        }
    ]

    storeSomeValsInMemory.splice(3 * 4, 0, ...pointlessOperation) // 3 is num elements per iteration of MSTOREing, 3*N == insert opp after N memory elements have been written
    
    // const numWordsToStash = 2

    // const stashInMemoryOp: EVMBytecode = dynamicStashMemoryInStack(numWordsToStash)
    // log.debug(`the slice of bytecode being used to stash memory into the stack is: \n${formatBytecode(stashInMemoryOp)}\n   Which is length:${stashInMemoryOp.length}`)
    // const unstashFromMemoryOp: EVMBytecode = dynamicUnstashMemoryFromStack(numWordsToStash)
    // log.debug(`the slice of bytecode being used to unstash memory into the stack is: \n${formatBytecode(unstashFromMemoryOp)}\n   Which is length:${unstashFromMemoryOp.length}`)


    const memoryIndexToModify: number = 2
    const numWordsToModify: number = 2

    // const pushMemoryIndexToTempModify: EVMOpcodeAndBytes = {
    //     opcode: Opcode.PUSH1,
    //     consumedBytes: new BigNumber(memoryIndexToModify).toBuffer('B',1)
    // }

    // let temporaryMemoryModification: EVMBytecode = [
    //     {
    //         opcode: Opcode.POP,
    //         consumedBytes: undefined
    //     },

    //     pushMemoryIndexToTempModify
    // ].concat(
    //     stashInMemoryOp
    // ).concat(
    //     overwriteNWordsInMemoryWithOffset(2, 2)
    // ).concat(
    //     unstashFromMemoryOp
    // ).concat( // this one just so memory prints at the end
    //     {
    //         opcode: Opcode.POP,
    //         consumedBytes: undefined
    //     }
    // )

    let stashModifyUnstash: EVMBytecode = [
        ...staticStashMemoryInStack(memoryIndexToModify, numWordsToModify),
        ...overwriteNWordsInMemoryWithOffset(2, 2),
        ...staticUnstashMemoryFromStack(memoryIndexToModify, numWordsToModify)
    ]

    let replaceMap: Map<EVMOpcode, EVMBytecode> = new Map<EVMOpcode, EVMBytecode>().set(
        Opcode.POP,
        [
            { // retain the POP we will be replacing so that the PUSH POP still works
                opcode: Opcode.POP, consumedBytes: undefined
            },
            ...stashModifyUnstash
        ]
    )

    opcodeWhitelist = new OpcodeWhitelistImpl(whitelistedOpcodes)
    replacer = new OpcodeReplacerImpl(
      stateManagerAddress,
      replaceMap
    )
    transpiler = new TranspilerImpl(opcodeWhitelist, replacer)
    evmUtil = new EvmIntrospectionUtilImpl()
  })

  it('Memory operations between a stash and unstash operation should not have any effect', async () => {
    const memoryOperatingBytecodeBuf: Buffer = bytecodeToBuffer(storeSomeValsInMemory)
    log.debug(
        `Running the memory modifying non transpiled code first, it is: \n${formatBytecode(
            storeSomeValsInMemory
        )}`
      )
    const preTransResult = await evmUtil.getExecutionResult(memoryOperatingBytecodeBuf)
    log.debug(`pre transpilation execution result is: ${JSON.stringify(preTransResult)}`)

    const transpilation = transpiler.transpile(memoryOperatingBytecodeBuf) as SuccessfulTranspilation
    const transpiledMemoryOperations: Buffer = transpilation.bytecode
    log.debug(
        `Running the memory modifying TRANSPILED code second (now), it is: \n${formatBytecode(
            bufferToBytecode(transpilation.bytecode)
        )}`
      )
    const postTransResult = await evmUtil.getExecutionResult(transpiledMemoryOperations)

    // const executionResults = await evmUtil.getExecutionResultComparison(memoryOperatingBytecodeBuf, transpiledMemoryOperations)
    // executionResults.resultsDiffer.should.be.false
  })

})