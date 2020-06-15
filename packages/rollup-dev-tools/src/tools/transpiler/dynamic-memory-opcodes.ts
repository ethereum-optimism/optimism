/* External Imports */
import { Opcode, EVMBytecode, Address } from '@eth-optimism/rollup-core'
import { getLogger, hexStrToBuf } from '@eth-optimism/core-utils'
import {
  getSWAPNOp,
  getPUSHIntegerOp,
  getDUPNOp,
  pushMemoryOntoStack,
  getPUSHBuffer,
  storeStackInMemory,
  getPUSHOpcode,
} from './helpers'
import * as abi from 'ethereumjs-abi'

const log = getLogger(`call-type-replacement-gen`)

export const ovmCALLName: string = 'ovmCALL'
export const ovmSTATICCALLName: string = 'ovmSTATICCALL'
export const ovmDELEGATECALLName: string = 'ovmDELEGATECALL'
export const ovmEXTCODECOPYName: string = 'ovmEXTCODECOPY'

/**
 * This replaces the CALL Opcode with a CALL to our ExecutionManager.
 * Notably, this:
 *  * Assumes the proper stack is in place to do the un-transpiled CALL
 *  * Replaces the call address in the stack with the executionManagerAddress
 *  * Safely prepends method id and arguments to CALL argument memory
 *  * Updates CALL memory index and length to account for prepended arguments
 *
 * @param executionManagerAddress The address of the Execution Manager contract.
 * @param ovmCALLFunctionName (ONLY USE FOR TESTING) The function name in the Execution Manager to handle DELEGTECALLs.
 */
export const getCALLReplacement = (
  executionManagerAddress: Address,
  ovmCALLFunctionName: string = ovmCALLName
): EVMBytecode => {
  return getCallTypeReplacement(executionManagerAddress, ovmCALLFunctionName, 3)
}

/**
 * This replaces the STATICCALL Opcode with a CALL to our ExecutionManager.
 * Notably, this:
 *  * Assumes the proper stack is in place to do the un-transpiled CALL
 *  * Replaces the call address in the stack with the executionManagerAddress
 *  * Safely prepends method id and arguments to STATICCALL argument memory
 *  * Updates STATICCALL memory index and length to account for prepended arguments
 *
 * @param executionManagerAddress The address of the Execution Manager contract.
 * @param ovmSTATICCALLFunctionName (ONLY USE FOR TESTING) The function name in the Execution Manager to handle DELEGTECALLs.
 */
export const getSTATICCALLReplacement = (
  executionManagerAddress: Address,
  ovmSTATICCALLFunctionName: string = ovmSTATICCALLName
): EVMBytecode => {
  return getCallTypeReplacement(
    executionManagerAddress,
    ovmSTATICCALLFunctionName,
    2
  )
}

/**
 * This replaces the DELEGATECALL Opcode with a CALL to our ExecutionManager.
 * Notably, this:
 *  * Assumes the proper stack is in place to do the un-transpiled CALL
 *  * Replaces the call address in the stack with the executionManagerAddress
 *  * Safely prepends method id and arguments to DELEGATECALL argument memory
 *  * Updates DELEGATECALL memory index and length to account for prepended arguments
 *
 * @param executionManagerAddress The address of the Execution Manager contract.
 * @param ovmDELEGATECALLFunctionName (ONLY USE FOR TESTING) The function name in the Execution Manager to handle DELEGTECALLs.
 */
export const getDELEGATECALLReplacement = (
  executionManagerAddress: Address,
  ovmDELEGATECALLFunctionName: string = ovmDELEGATECALLName
): EVMBytecode => {
  return getCallTypeReplacement(
    executionManagerAddress,
    ovmDELEGATECALLFunctionName,
    2
  )
}

/**
 * This replaces CALL-type Opcodes with CALLs to our ExecutionManager.
 * Notably, this:
 *  * Assumes the input stack is: [(PC of replaced opcode), ...[CALL arguments], ...]
 *  * Replaces the call address in the stack with the executionManagerAddress
 *  * Safely prepends method id and arguments to CALL argument memory
 *  * Updates CALL memory index and length to account for prepended arguments
 *
 * @param executionManagerAddress The address of the Execution Manager contract.
 * @param executionManagerCALLMethodName The function name in the Execution Manager to handle CALLs.
 * @param stackPositionOfCallArgsMemOffset The position on the stack of the CALL's arguments memory offset.  0-indexed.
 */
const getCallTypeReplacement = (
  executionManagerAddress: Address,
  executionManagerCALLMethodName: string,
  stackPositionOfCallArgsMemOffset: number // expected to be 2 or 3 depending on value presence
): EVMBytecode => {
  // we're gonna MSTORE methodId + addr
  const callMemoryWordsToPrepend: number = 1 + 1 // NOTE: if we needed to pass the call value in the future alongside addr, we would increment this
  const callMemoryBytesToPrepend: number = 32 * callMemoryWordsToPrepend

  const totalCALLTypeStackArguments: number =
    stackPositionOfCallArgsMemOffset + 4 // 4 for retIndex, retLen, argIndex, argLen

  const methodData: Buffer = abi.methodID(executionManagerCALLMethodName, [])

  // first, we store the memory we're going to overwrite in order to prepend methodId and params to the stack so the original memory can be recovered.
  const op: EVMBytecode = [
    // we will subtract the number of words we will prepend to get the index of memory we're pushing to stack to recover later
    getPUSHIntegerOp(callMemoryBytesToPrepend),
    getDUPNOp(1 + 1 + stackPositionOfCallArgsMemOffset + 1), // dup modified calldata arg index so it can be stored there
    { opcode: Opcode.SUB, consumedBytes: undefined }, // do subtraction
    // actually push it to the stack
    ...pushMemoryOntoStack(callMemoryWordsToPrepend),
  ]

  // stack should now be [(index of mem pushed to stack), ...[mem words pushed to stack], (PC of replaced opcode), ...[CALL params], ...]
  // duplicate the four memory-related params from the original CALL to front of stack
  op.push(
    ...new Array(4).fill(
      getDUPNOp(1 + callMemoryWordsToPrepend + totalCALLTypeStackArguments + 1)
    )
  )

  // stack should now be [argOffstet, argLen, retOffst, retLength, (index of mem pushed to stack), ...[mem words pushed to stack], (PC of replaced opcode), ...[CALL params], ...]
  // where those first four memory params are un-modified from the pre-transpiled CALL.

  // now, we need to push the additional calldata to stack and store stack in memory.
  // NOTE: if we needed to pass the call value in the future alongside addr, we would dup that additional val to stack here.

  // dup the ADDR param from the initial stack. based on the expected stack above (and that PUSHBuffer), its index is 4 + 1 + callMemoryWordsToPrepend + 1 + 2
  op.push(getDUPNOp(callMemoryWordsToPrepend + 8))
  // PUSH the method data to stack
  op.push(getPUSHBuffer(methodData))

  // store the [methodId, stack args] words in memory. To do this, we DUP (index of mem pushed to stack), because that's what the storeStackInMemory() expects as first element.
  op.push(
    getDUPNOp(2 + 4 + 1), // EM CALL args offset is immediately after (previous DUPN and PUSHBuffer = 2) + (memory args = 4)
    ...storeStackInMemory(callMemoryWordsToPrepend)
  )
  // pop the index we were just using to store stack in memory
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })

  // at this point the stack should be [4 words of CALL memory arguments, EM CALL args offset, ...[words pulled from memory], ...original stack]
  // now that we have prepended the correct calldata to memory, we need to update the args length and offset appropriately to actually pass the prepended data.
  const numBytesForExtraArgs: number = 4 + 1 * 32 // methodId + (Num params )* 32 bytes/word // NOTE: if we want to pass another param down the line, increment this "1"
  op.push(
    //subtract it from the offset, should be the first thing on the stack
    getPUSHIntegerOp(numBytesForExtraArgs),
    getSWAPNOp(1),
    { opcode: Opcode.SUB, consumedBytes: undefined },
    // add it to the length, should be the second thing on the stack
    // swap from second to first
    getSWAPNOp(1),
    // add
    getPUSHIntegerOp(numBytesForExtraArgs),
    { opcode: Opcode.ADD, consumedBytes: undefined },
    // swap back from first to second
    getSWAPNOp(1)
  )
  // now we are ready to execute the call.  The memory-related args have already been set up, we just need to add the first three fields and execute.
  op.push(
    // value (0 ETH always!)
    getPUSHIntegerOp(0),
    // address
    getPUSHBuffer(hexStrToBuf(executionManagerAddress)),
    // Gas -- just use the original gas from abov
    getDUPNOp(9 + callMemoryWordsToPrepend), // 1 (address) + 1 (value) + 4 (memory args) + 1 (replacement index) + callMemoryWordsToPrepend (the preserved words themselves) + 1 (PC to return to) +1 (gas is element of a CALL stack)
    // CALL!
    {
      opcode: Opcode.CALL,
      consumedBytes: undefined,
    }
  )
  // now we have the success result at the top of the stack, so we swap it out to where it will be first after we put back the old memory and pop the original params.
  // this index should be:
  //  +1 for memory replacment index
  //  + callMemoryWordsToPrepend
  //  +1 for PC of replaced opcode
  //  + number of args to the original CALL-type
  op.push(
    getSWAPNOp(1 + callMemoryWordsToPrepend + 1 + totalCALLTypeStackArguments)
  )

  // we swapped with garbage stack which we no longer need since CALL has been executed, so POP
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })
  // now that the success result is out of the way we can return memory to original state, the index and words are first on stack now!
  op.push(...storeStackInMemory(callMemoryWordsToPrepend))
  // POP the index just used to store stack back in memory
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })
  // expected stack is now: [(PC of replaced opcode), ...[call args missing 1 element from the garbage POP above], (success), ...].  We need to preserve the PC, so swap it to right before the success.
  op.push(getSWAPNOp(totalCALLTypeStackArguments - 1))

  // lastly, POP all the original CALL params which were previously DUPed and modified appropriately.
  op.push(
    ...new Array(totalCALLTypeStackArguments - 1).fill({
      opcode: Opcode.POP,
      consumedBytes: undefined,
    })
  )

  return op
}

/**
 * This replaces EXTCODECOPY Opcode with a CALL to our ExecutionManager.
 * Notably, this:
 *  * Assumes the stack is: [(PPC to return to), ...[EXTCODECOPY untranspiled args], ...]
 *  * Stores memory to be modified to the stack
 *  * Safely stores method id and arguments to CALL argument memory
 *  * CALLs the specified ovmEXTCODECOPY function
 *  * Returns memory to its original pre-CALL state
 *
 * @param executionManagerAddress The address of the Execution Manager contract.
 * @param ovmEXTCODECOPYFunctionName (ONLY USE FOR TESTING) The function name in the Execution Manager to handle EXTCODECOPYs.
 */

export const getEXTCODECOPYReplacement = (
  executionManagerAddress: Address,
  ovmEXTCODECOPYFunctionName: string = ovmEXTCODECOPYName
) => {
  const methodData: Buffer = abi.methodID(ovmEXTCODECOPYFunctionName, [])
  const op: EVMBytecode = []

  // EXTCODECOPY params and execution do the following to the stack:
  // [addr, destOffset, offset, length, …], -> […]

  // We will only need to pass addr, index, length as calldata.
  // (destOffset is what the opcode writes to, so it is not passed to the X-Mgr but instead reflected in the CALL's retOffset)
  const numStackWordsToPass: number = 3
  // this is the number of memory words we'll be overwriting for the call.
  const callMemoryWordsToPass: number = 1 + numStackWordsToPass // 1 extra for methodId!

  // first, we push the memory we're gonna overwrite with calldata onto the stack, so that it may be parsed later.
  // to make sure there is not a collision between call and return data locations, we will store this AFTER (destOffset + length)
  op.push(
    getDUPNOp(5), // DUP length
    getDUPNOp(4), // DUP destOffset
    { opcode: Opcode.ADD, consumedBytes: undefined }, // add them to get where we expect to store
    ...pushMemoryOntoStack(callMemoryWordsToPass)
  )
  // Now, the stack should be [(index of memory to replace), ...[pushed memory to swap back], (PC to return to), ...[original EXTCODECOPY args], ...]

  // We now store the stack params needed by the execution manager into memory to pass as calldata.
  // ovmEXTCODSIZE expects the following raw bytes as parameters:
  // *       [methodID (bytes4)]
  // *       [targetOvmContractAddress (address as bytes32)]
  // *       [index (uint (32)]
  // *       [length (uint (32))]
  // so we will push thse in reverse order.

  const indexOfOriginalStack: number = 1 + callMemoryWordsToPass + 1
  op.push(
    // the final params of addition here (+0, +1, +2) account for the increased stack caused by each preceeding DUP
    getDUPNOp(indexOfOriginalStack + 4 + 0), // length
    getDUPNOp(indexOfOriginalStack + 3 + 1), // offset
    getDUPNOp(indexOfOriginalStack + 1 + 2), // addr
    getPUSHBuffer(methodData), // methodId
    getDUPNOp(1 + numStackWordsToPass + 1), // the mem index to store calldata -- it's right after all the words to store we just pushed
    ...storeStackInMemory(1 + numStackWordsToPass), // +1 for methodId
    {
      // pop the storage index as it was DUPed above
      opcode: Opcode.POP,
      consumedBytes: undefined,
    }
  )
  // the stack should now be [(mem index to recover memory to), ...[memory words to recover], (PC to return to), ...[original EXTCODECOPY args], ...]
  // Now we need to set up the CALL!
  const numBytesForCalldata: number = 4 + numStackWordsToPass * 32 // methodId + (Num params)* 32 bytes/word
  // CALL expects:
  // [gas, addr, value, argsOffset, argsLength, retOffset, retLength, …] -> [success, …]
  op.push(
    getDUPNOp(0 + 1 + callMemoryWordsToPass + 4 + 1), // retLength is same as original stack's `len` (4th element)
    getDUPNOp(1 + 1 + callMemoryWordsToPass + 2 + 1), // retOffset is second stack item of original stack
    getPUSHIntegerOp(numBytesForCalldata), // argsLen
    // argsOffset is wherever we stored the params in memory, with added 32-4 = 28 bytes of 0s when methodId was MSTORE'd which we don't want to pass
    getDUPNOp(3 + 1), // three elements were just pushed, next is th index we stored at
    getPUSHIntegerOp(28),
    { opcode: Opcode.ADD, consumedBytes: undefined },
    getPUSHIntegerOp(0), // value is always 0!
    getPUSHBuffer(hexStrToBuf(executionManagerAddress)), // X mgr address
    getPUSHIntegerOp(10000000), // random sufficient amount of gas
    // execute the call!
    {
      opcode: Opcode.CALL,
      consumedBytes: undefined,
    },
    // POP success, x mgr should never fail here.
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    }
  )
  // Cleanup time is all that's left!
  op.push(
    ...storeStackInMemory(callMemoryWordsToPass), // recover the original memory we pushed the stack
    // POP the index of stored memory, no longer needed.
    { opcode: Opcode.POP, consumedBytes: undefined },
    // stack should now be [(PC to return to), ...[EXTCODECOPY args], ...].  We need to preserve it so swap it to the end
    getSWAPNOp(4),
    // pop the rest of the args which have now served their purpose.  RIP
    ...new Array(4).fill({
      opcode: Opcode.POP,
      consumedBytes: undefined,
    })
  )
  return op
}
