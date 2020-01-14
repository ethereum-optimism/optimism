/* External Imports */
import {
    Opcode,
    EVMOpcode,
    EVMOpcodeAndBytes,
    EVMBytecode,
    isValidOpcodeAndBytes,
    Address,
    formatBytecode,
  } from '@pigi/rollup-core'
  import {
    bufToHexString,
    remove0x,
    getLogger,
    isValidHexAddress,
    hexStrToBuf,
    BigNumber,
  } from '@pigi/core-utils'
  import { ADDRCONFIG } from 'dns'
  import { POINT_CONVERSION_HYBRID } from 'constants'
import { getSWAPNOp, getPUSHIntegerOp, getDUPNOp, pushMemoryOntoStack, getPUSHBuffer, storeStackInMemory } from './memory-substitution'
import * as abi from 'ethereumjs-abi'

  
  const log = getLogger(`call-type-replacement-gen`)

  export const getCallTypeReplacement = (
    proxyAddress: Address,
    methodName: string,
    numStackArgumentsToPass: number
  ): EVMBytecode => {
      const numMemoryWordsToPreserve: number = 1 + numStackArgumentsToPass
      const numMemoryBytesToPreserve: number = 32 * numMemoryWordsToPreserve

      const methodData: Buffer = abi.methodID(methodName, [])

      // first, we store the memory we're going to overwrite in order to prepend methodId and params to the stack so the original memory can be recovered.
      let op: EVMBytecode = [
          getDUPNOp(numStackArgumentsToPass + 1), // dup modified calldata arg index so it can be stored there
          // subtract the number of words we will prepend to get the index of memory we're pushing to stack to recover later
          getPUSHIntegerOp(numMemoryBytesToPreserve),
          getSWAPNOp(1),
          { opcode: Opcode.SUB, consumedBytes: undefined},
          // actually push it to the stack
          ...pushMemoryOntoStack(numMemoryWordsToPreserve),
      ]

      // duplicates pre-stashed stack
      for (let i: number = 0; i < numStackArgumentsToPass + 4; i++) {
        op = [...op,
            getDUPNOp(1 + numMemoryWordsToPreserve + numStackArgumentsToPass + 4) 
        ]
      }
      // PUSH the method data to stack
      op = [
          ...op,
          getPUSHBuffer(methodData)
      ]
      // duplicate the mem index to store prepended calldata to
      op = [
          ...op, 
          getDUPNOp(1 + numStackArgumentsToPass + 4 + 1) 
        ]

        // debug push-pop. TODO delete this
        op = [...op, getPUSHIntegerOp(0), {opcode: Opcode.POP, consumedBytes: undefined}]
      // store the [methodId, stack args] words in memory at the index which we just DUP'ed
      op = [
          ...op,
          ...storeStackInMemory(1 + numStackArgumentsToPass)
      ]
      // pop the index we were just using to store stack in memory
      op = [
          ...op,
          {opcode: Opcode.POP, consumedBytes: undefined}
      ] 

      // at this point the stack should be [last 4 args of memory stuff, memory index of where to re-write, ...[words pulled from memory], ...original stack]
      // now that we have prepended the correct calldata, we need to update the args length and offset appropriately to actually pass the prepended data.
    const numBytesForExtraArgs: number = 4 + 32 * numStackArgumentsToPass // methodId + Num params * 32 bytes/word
op = [
    ...op,
//subtract it from the offset, should be the first thing on the stack
            getPUSHIntegerOp(numBytesForExtraArgs),
          { opcode: Opcode.SWAP1, consumedBytes: undefined},
          { opcode: Opcode.SUB, consumedBytes: undefined},
// add it to the length, should be the second thing on the stack
          // swap from second to first
          getSWAPNOp(1),
          // add
          getPUSHIntegerOp(numBytesForExtraArgs),
          {opcode: Opcode.ADD, consumedBytes: undefined},
          // swap back from first to second
          getSWAPNOp(1)
]
 // now we are ready to execute the call.  The memory-related args have already been set up, we just need to add the first three fields and execute.
op = [
    ...op,
    // value (0 ETH always!)
    getPUSHIntegerOp(0),
      // address
      getPUSHBuffer(hexStrToBuf(proxyAddress)),
      // Gas
      {
        opcode: Opcode.PUSH32,
        consumedBytes: Buffer.from('00'.repeat(16) + 'ff'.repeat(16), 'hex'),
      },
      // execute the call!
      {
        opcode: Opcode.CALL,
        consumedBytes: undefined,
      },
]
// now we have the success result at the top of the stack, so we swap it out to where it will be first after we put back the old memory and pop the original params.
// this index should be (1 for memory replacment index + numMemoryWordsToPreserve + numStackArgumentsToPass + 4 for memory offset and calldata for arg vals and return vals))
    op.push(
        getSWAPNOp(1 + numMemoryWordsToPreserve + numStackArgumentsToPass + 4)
    )

    // we swapped with garbage stack which we no longer need since CALL has been executed, so POP
    op.push({opcode: Opcode.POP, consumedBytes: undefined})
    // now that the success result is out of the way we can return memory to original state, the index and words are first on stack now!
    op = [
        ...op,
        ...storeStackInMemory(numMemoryWordsToPreserve)
    ]

    // lastly, POP all the original CALL params which were previously DUPed and modified appropriately.
    op = [...op,
        ...new Array(1 + numStackArgumentsToPass + 4 - 1).fill({opcode: Opcode.POP, consumedBytes: undefined})
    ]


    return op
  }