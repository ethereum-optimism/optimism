/* External Imports */
import { Opcode, EVMBytecode, Address } from '@eth-optimism/rollup-core'
import { getLogger, hexStrToBuf } from '@eth-optimism/core-utils'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Internal Imports */
import {
  getSWAPNOp,
  getPUSHIntegerOp,
  getDUPNOp,
  pushMemoryOntoStack,
  getPUSHBuffer,
  storeStackInMemory,
} from './helpers'
import { BIG_ENOUGH_GAS_LIMIT } from './'

const log = getLogger(`contract-creation-replacement-gen`)

export const ovmCREATEName = 'ovmCREATE'
export const ovmCREATE2Name = 'ovmCREATE2'

/**
 * This replaces CREATE Opcode with a CALL to our ExecutionManager.
 * Notably, this:
 *  * Assumes the proper memory for create and a stack of (PC to return to), (untranspiled CREATE args), ...
 *  * Stores memory that will be modified during the proxy operation to the stack
 *  * Safely stores ovmCREATE method id and arguments to memory so it can be passed with the proxy CALL.
 *  * CALLs the specified ovmCREATE function
 *  * Pushes the returned created address to the stack.
 *  * Returns memory to its original pre-CALL state and cleans up the stack to what a normal CREATE would do.
 *
 * @param executionManagerAddress The address of the Execution Manager contract.
 * @param ovmCREATEFunctionName (ONLY USED FOR TESTING) The function name in the Execution Manager to handle CREATEs.
 */
export const getCREATESubstitute = (
  executionManagerAddress: Address,
  ovmCREATEFunctionName: string = ovmCREATEName
): EVMBytecode => {
  // CREATE params and execution do the following to the stack:
  // [value, offset, length, ...] --> 	[addr, ...]
  // Where offset and length are memory indices of the initcode.
  // additionally we expect the PC to JUMP back to to be preserved, so input stack on entering this function's bytcode is [(PC to jump back to), value, offset, length, ...]

  // The execution manager expects th following calldata: (variable-length bytes)
  //  *       [methodID (bytes4)]
  //  *       [ovmInitcode (bytes (variable length))]
  // so, we're gonna MSTORE methodId prepended to the original CREATE offset and length.

  const callMemoryWordsToPrepend: number = 1 // NOTE: if we needed to pass the call value in the future alongside addr, we would increment this
  const callMemoryBytesToPrepend: number = 32 * callMemoryWordsToPrepend

  const methodId: Buffer = ethereumjsAbi.methodID(ovmCREATEFunctionName, [])

  // First, we store the memory we're going to overwrite in order to prepend methodId and params to the stack so the original memory can be recovered.
  // We will use this same memory for recovering the returned created Addr after the call.  So it will be referred to as retOffset in these comments.
  const op: EVMBytecode = [
    // we will subtract the number of words we will prepend to get the index of memory we're pushing to stack to recover later (this will be reused as retOffset)
    getPUSHIntegerOp(callMemoryBytesToPrepend),
    getDUPNOp(4), // dup memory offset of initcode, this is expected at index 3, after what we just pushed -> 4
    { opcode: Opcode.SUB, consumedBytes: undefined }, // do subtraction
    // actually push it to the stack
    ...pushMemoryOntoStack(callMemoryWordsToPrepend),
  ]

  // stack should now be [retOffset, ...[mem words pushed to stack], (PC to return to), ...[value, offset, length, ...]]]
  // duplicate the two memory-related params from the original CREATE to front of stack
  op.push(
    ...new Array(2).fill(
      getDUPNOp(1 + callMemoryWordsToPrepend + 1 + 3) // this will DUP length then offset
    )
  )

  // stack should now be [offset, length, retOffset, ...[mem words pushed to stack], (PC to return to), ...[value, offset, length, ...]]
  // where the first two memory params are un-modified from the pre-transpiled CALL.

  // now, we need to store the prepended calldata in memory right before the original offset.
  // NOTE: if we needed to pass the call value in the future alongside addr, we would dup that additional val to stack here and MSTORE it.

  // PUSH the method data to stack
  op.push(getPUSHBuffer(methodId))

  // store the methodId words in memory. To do this, we DUP retOffset, because that's what the storeStackInMemory() expects as first element.
  op.push(
    getDUPNOp(4), // Stack is [methodId, offset, length, retOffset, ... ...]
    ...storeStackInMemory(callMemoryWordsToPrepend)
  )
  // pop the index we were just using to store stack in memory
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })

  // at this point the stack should be back to [offset, length, retOffset, ...[mem words pushed to stack], (PC to return to), ...[value, offset, length, ...]]
  // now that we have prepended the correct calldata to memory, we need to update the args length and offset appropriately to actually pass the prepended data.
  const numBytesForExtraArgs: number = 4 + 0 * 32 // methodId + (Num params )* 32 bytes/word // NOTE: if we want to pass value param down the line, increment this "0"
  op.push(
    //subtract number of additional bytes from the initial offseet, should be the first thing on the stack
    getPUSHIntegerOp(numBytesForExtraArgs),
    getSWAPNOp(1),
    { opcode: Opcode.SUB, consumedBytes: undefined },
    // add number of additional bytes to the initial length, should be the second thing on the stack
    // swap from [length, offset, ...] to [offset, length, ...]
    getSWAPNOp(1),
    // add
    getPUSHIntegerOp(numBytesForExtraArgs),
    { opcode: Opcode.ADD, consumedBytes: undefined },
    // swap initcodeLength back so stack is [length, offset, ...] again
    getSWAPNOp(1)
  )
  // CALL expects:
  // The argOffset and argLength have already been set up, but we need retOffset and retLength above those (3rd and 4th stack elements)
  // we'll overwrite the methodId word to accomplish this, since it's no longer needed once the call executes.
  // This means retOffset, and retLength = 32 since execution manager will return addr as 32-byte big-endian.
  const retLength: number = 32
  op.push(
    getPUSHIntegerOp(retLength), // push retLength, stack should now be [retLength, argLength, argOffset, retOffset...]
    getSWAPNOp(2), // swap into third element, stack should now be [argLength, argOffset, 32, retOffset...]
    getDUPNOp(4), // dup retOffset, stack should now be [retOffset, argLength, argOffset, 32, retOffset...]
    getSWAPNOp(2) // swap into third element, stack should now line up with last 4 inputs to CALL
  )
  // now, we just need to add the first three fields and execute
  op.push(
    // value (0 ETH always!)
    getPUSHIntegerOp(0),
    // address
    getPUSHBuffer(hexStrToBuf(executionManagerAddress)),
    getPUSHIntegerOp(BIG_ENOUGH_GAS_LIMIT),
    // CALL!
    {
      opcode: Opcode.CALL,
      consumedBytes: undefined,
    },
    // POP success--should always be successful
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    }
  )
  // stack should now be [retOffset, ...[mem pushed to stack], (PC to return to), ...[value, offset, length, ...]]
  // We need to pull the addr from memory at retOffset before overwriting it back to the original memory.
  op.push(getDUPNOp(1), {
    opcode: Opcode.MLOAD,
    consumedBytes: undefined,
  })

  // now we have the addr result at the top of the stack, so we swap it out to where it will be first after we put back the old memory and pop the original params.
  // this index should be (1 for memory replacment index + callMemoryWordsToPrepend + 4 for original [(PC to return to), value, offset, length])
  op.push(getSWAPNOp(1 + callMemoryWordsToPrepend + 4))

  // we swapped with garbage stack which we no longer need since CALL has been executed, so POP
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })
  // now that the success result is out of the way we can return memory to original state, the index and words are first on stack now!
  op.push(...storeStackInMemory(callMemoryWordsToPrepend))
  // POP the index used to storeStackInMemeorry
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })

  // now, the stack should be [(PC to JUMP back to), value, offset, addr] (note that length was swapped and popped with create)
  // swap the PC with value so that we preserve it
  op.push(getSWAPNOp(2))
  // lastly, POP the value and offset
  op.push(
    ...new Array(2).fill({
      opcode: Opcode.POP,
      consumedBytes: undefined,
    })
  )

  return op
}

/**
 * This replaces CREATE2 Opcode with a CALL to our ExecutionManager.
 * Notably, this:
 *  * Assumes the proper memory for create and a stack of (PC to return to), (untranspiled CREATE2 args), ...
 *  * Stores memory that will be modified during the proxy operation to the stack
 *  * Safely stores ovmCREATE2 method id and salt to memory so it can be passed with the proxy CALL.
 *  * CALLs the specified ovmCREATE2 function
 *  * Pushes the returned created address to the stack.
 *  * Returns memory to its original pre-CALL state and cleans up the stack to what a normal CREATE2 would do.
 *
 * @param executionManagerAddress The address of the Execution Manager contract.
 * @param ovmCREATE2FunctionName The function name in the Execution Manager to handle CREATE2s.
 */
export const getCREATE2Substitute = (
  executionManagerAddress: Address,
  ovmCREATE2FunctionName: string = ovmCREATE2Name
): EVMBytecode => {
  // CREATE2 params and execution do the following to the stack:
  // [value, offset, length, salt, ...] --> 	[addr, ...]
  // Where offset and length are memory indices of the initcode.
  // additionally we expect the PC to JUMP back to to be preserved, so input stack on entering this function's bytcode is [(PC to jump back to), value, offset, length, salt, ...]

  // The execution manager expects th following calldata: (variable-length bytes)
  // *       [methodID (bytes4)]
  // *       [salt (bytes32)]
  // *       [ovmInitcode (bytes (variable length))]
  // so, we're gonna MSTORE methodId and salt prepended to the original CREATE2 offset and length.

  const callMemoryWordsToPrepend: number = 2 // NOTE: if we needed to pass the call value in the future alongside addr, we would increment this
  const callMemoryBytesToPrepend: number = 32 * callMemoryWordsToPrepend

  const methodId: Buffer = ethereumjsAbi.methodID(ovmCREATE2FunctionName, [])

  // First, we store the memory we're going to overwrite in order to prepend methodId and params to the stack so the original memory can be recovered.
  // We will use this same memory for recovering the returned created Addr after the call.  So it will be referred to as retOffset in these comments.
  const op: EVMBytecode = [
    // we will subtract the number of words we will prepend to get the index of memory we're pushing to stack to recover later (this will be reused as retOffset)
    getPUSHIntegerOp(callMemoryBytesToPrepend),
    getDUPNOp(4), // dup memory offset of initcode, this is expected at index 3, after what we just pushed -> 4
    { opcode: Opcode.SUB, consumedBytes: undefined }, // do subtraction
    // actually push it to the stack
    ...pushMemoryOntoStack(callMemoryWordsToPrepend),
  ]

  // stack should now be [retOffset, ...[mem words pushed to stack], (pc to return to), ...[value, offset, length, salt, ...]]]
  // duplicate the two memory-related params from the original CREATE to front of stack
  op.push(
    ...new Array(2).fill(
      getDUPNOp(1 + callMemoryWordsToPrepend + 1 + 3) // this will DUP length then offset
    )
  )

  // stack should now be [offset, length, retOffset, ...[mem words pushed to stack], ...[value, offset, length, salt, ...]]
  // where the first two memory params are un-modified from the pre-transpiled CALL.

  // now, we need to store the prepended calldata in memory right before the original offset.
  // NOTE: if we needed to pass the call value in the future alongside addr, we would dup that additional val to stack here and MSTORE it.

  // DUP the salt
  op.push(getDUPNOp(2 + 1 + callMemoryWordsToPrepend + 1 + 4)) // see "stack should now be" section above for justification of this indexing
  // PUSH the method data to stack
  op.push(getPUSHBuffer(methodId))

  // store the methodId words in memory. To do this, we DUP retOffset, because that's what the storeStackInMemory() expects as first element.
  op.push(
    getDUPNOp(5), // Stack is [methodId, salt, offset, length, retOffset, ... ...]
    ...storeStackInMemory(callMemoryWordsToPrepend)
  )
  // pop the index we were just using to store stack in memory
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })

  // at this point the stack should be back to [offset, length, retOffset, ...[mem words pushed to stack], (PC to return to), ...[value, offset, length, salt, ...]]
  // now that we have prepended the correct calldata to memory, we need to update the args length and offset appropriately to actually pass the prepended data.
  const numBytesForExtraArgs: number = 4 + 1 * 32 // methodId + (Num params AKA salt)* 32 bytes/word
  op.push(
    // subtract number of additional bytes from the initial offseet, should be the first thing on the stack
    getPUSHIntegerOp(numBytesForExtraArgs),
    getSWAPNOp(1),
    { opcode: Opcode.SUB, consumedBytes: undefined },
    // add number of additional bytes to the initial length, should be the second thing on the stack
    // swap from [length, offset, ...] to [offset, length, ...]
    getSWAPNOp(1),
    // add
    getPUSHIntegerOp(numBytesForExtraArgs),
    { opcode: Opcode.ADD, consumedBytes: undefined },
    // swap initcodeLength back so stack is [length, offset, ...] again
    getSWAPNOp(1)
  )
  // CALL expects:
  // The argOffset and argLength have already been set up, but we need retOffset and retLength above those (3rd and 4th stack elements)
  // we'll overwrite the methodId word to accomplish this, since it's no longer needed once the call executes.
  // This means retOffset == retOffset, and retLength = 32 since execution manager will return addr as 32-byte big-endian.
  const retLength: number = 32
  op.push(
    getPUSHIntegerOp(retLength), // push retLength, stack should now be [retLength, argLength, argOffset, retOffset...]
    getSWAPNOp(2), // swap into third element, stack should now be [argLength, argOffset, 32, retOffset...]
    getDUPNOp(4), // dup retOffset, stack should now be [retOffset, argLength, argOffset, 32, retOffset...]
    getSWAPNOp(2) // swap into third element, stack should now line up with last 4 inputs to CALL
  )
  // now, we just need to add the first three fields and execute
  op.push(
    // value (0 ETH always!)
    getPUSHIntegerOp(0),
    // address
    getPUSHBuffer(hexStrToBuf(executionManagerAddress)),
    // gas TODO add sufficient_gas_constant
    getPUSHIntegerOp(100010001001),
    // CALL!
    {
      opcode: Opcode.CALL,
      consumedBytes: undefined,
    },
    // POP success--should always be successful
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    }
  )
  // stack should now be [retOffset, ...[mem pushed to stack], (PC to return to), ...[value, offset, length, salt, ...]]
  // We need to pull the addr from memory at retLength before overwriting it back to the original memory.
  op.push(getDUPNOp(1), {
    opcode: Opcode.MLOAD,
    consumedBytes: undefined,
  })

  // now we have the addr result at the top of the stack, so we swap it out to where it will be first after we put back the old memory and pop the original params.
  // this index should be (1 for memory replacment index + callMemoryWordsToPrepend + 5 for original [(PC to return to), value, offset, length, salt])
  op.push(getSWAPNOp(1 + callMemoryWordsToPrepend + 5))

  // we swapped with garbage stack which we no longer need since CALL has been executed, so POP
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })
  // now that the success result is out of the way we can return memory to original state, the index and words are first on stack now!
  op.push(...storeStackInMemory(callMemoryWordsToPrepend))
  // POP the index used by storeStackInMemory
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })
  // stack should now be  [(PC to return to), value, offset, length, (addr of CREATE2ed account)] (note salt was swapped and popped for addr above)
  // SWAP (PC to return to) to preserve it as first element
  op.push(getSWAPNOp(3))

  // POP the remaining value, offset, length
  op.push(
    ...new Array(3).fill({
      opcode: Opcode.POP,
      consumedBytes: undefined,
    })
  )

  return op
}
