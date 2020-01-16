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
import {
  getSWAPNOp,
  getPUSHIntegerOp,
  getDUPNOp,
  pushMemoryOntoStack,
  getPUSHBuffer,
  storeStackInMemory,
  pushMemoryAtIndexOntoStack,
} from './memory-substitution'
import * as abi from 'ethereumjs-abi'
import EVM from 'ethereumjs-vm/dist/evm/evm'

const log = getLogger(`call-type-replacement-gen`)

export const getCallTypeReplacement = (
  proxyAddress: Address,
  methodName: string,
  hasVALUEStackElement: boolean // 3 or 4 depending on value presence
): EVMBytecode => {
  // we're gonna MSTORE methodId + addr
  const numMemoryWordsToPreserve: number = 1 + 1 // NOTE: if we needed to pass the call value in the future alongside addr, we would increment this
  const numMemoryBytesToPreserve: number = 32 * numMemoryWordsToPreserve
  const numStackArgumentsBeforeMemoryArguments: number = hasVALUEStackElement
    ? 3
    : 4
  const totalCALLTypeStackArguments: number =
    numStackArgumentsBeforeMemoryArguments + 4 // 4 for retIndex, retLen, argIndex, argLen

  const methodData: Buffer = abi.methodID(methodName, [])

  // first, we store the memory we're going to overwrite in order to prepend methodId and params to the stack so the original memory can be recovered.
  let op: EVMBytecode = [
    getDUPNOp(numStackArgumentsBeforeMemoryArguments + 1), // dup modified calldata arg index so it can be stored there
    // subtract the number of words we will prepend to get the index of memory we're pushing to stack to recover later
    getPUSHIntegerOp(numMemoryBytesToPreserve),
    getSWAPNOp(1),
    { opcode: Opcode.SUB, consumedBytes: undefined },
    // actually push it to the stack
    ...pushMemoryOntoStack(numMemoryWordsToPreserve),
  ]

  // stack should now be [(index of mem pushed to stack), ...[mem words pushed to stack], ...[original stack]]]
  // duplicate the four memory-related params from the original CALL to front of stack
  op = [
    ...op,
    ...new Array(4).fill(
      getDUPNOp(1 + numMemoryWordsToPreserve + totalCALLTypeStackArguments)
    ),
  ]

  // stack should now be [argOffstet, argLen, retOffst, retLength, (index of mem pushed to stack), ...[mem words pushed to stack], ...[original stack]]]
  // where those first four memory params are un-modified from the pre-transpiled CALL.

  // now, we need to push the additional calldata to stack and store stack in memory.
  // NOTE: if we needed to pass the call value in the future alongside addr, we would dup that additional val to stack here.

  // dup the ADDR param from the initial stack. based on the expected stack above (and that PUSHBuffer), its index is 4 + 1 + numMemoryWordsToPreserve + 2
  op = [...op, getDUPNOp(numMemoryWordsToPreserve + 7)]
  // PUSH the method data to stack
  op = [...op, getPUSHBuffer(methodData)]

  // store the [methodId, stack args] words in memory. To do this, we DUP (index of mem pushed to stack), because that's what the storeStackInMemory() expects as first element.
  op = [
    ...op,
    getDUPNOp(2 + 4 + 1), // index is immediately after (previous DUPN and PUSHBuffer = 2) + (memory args = 4)
    ...storeStackInMemory(numMemoryWordsToPreserve),
  ]
  // pop the index we were just using to store stack in memory
  op = [...op, { opcode: Opcode.POP, consumedBytes: undefined }]

  // at this point the stack should be [last 4 args of memory stuff, memory index of where to re-write, ...[words pulled from memory], ...original stack]
  // now that we have prepended the correct calldata to memory, we need to update the args length and offset appropriately to actually pass the prepended data.
  const numBytesForExtraArgs: number = 4 + 1 * 32 // methodId + (Num params )* 32 bytes/word // NOTE: if we want to pass another param down the line, increment this "1"
  op = [
    ...op,
    //subtract it from the offset, should be the first thing on the stack
    getPUSHIntegerOp(numBytesForExtraArgs),
    { opcode: Opcode.SWAP1, consumedBytes: undefined },
    { opcode: Opcode.SUB, consumedBytes: undefined },
    // add it to the length, should be the second thing on the stack
    // swap from second to first
    getSWAPNOp(1),
    // add
    getPUSHIntegerOp(numBytesForExtraArgs),
    { opcode: Opcode.ADD, consumedBytes: undefined },
    // swap back from first to second
    getSWAPNOp(1),
  ]
  // now we are ready to execute the call.  The memory-related args have already been set up, we just need to add the first three fields and execute.
  op = [
    ...op,
    // value (0 ETH always!)
    getPUSHIntegerOp(0),
    // address
    getPUSHBuffer(hexStrToBuf(proxyAddress)),
    // Gas -- just use the original gas from abov
    getDUPNOp(8 + numMemoryWordsToPreserve), // 1 (address) + 1 (value) + 4 (memory args) + 1 (replacement index) + numMemoryWordsToPreserve (the preserved words themselves) + 1 (gas was first element of original stack)
    // CALL!
    {
      opcode: Opcode.CALL,
      consumedBytes: undefined,
    },
  ]
  // now we have the success result at the top of the stack, so we swap it out to where it will be first after we put back the old memory and pop the original params.
  // this index should be (1 for memory replacment index + numMemoryWordsToPreserve + numStackArgumentsToPass + 4 for memory offset and calldata for arg vals and return vals))
  op.push(
    getSWAPNOp(1 + numMemoryWordsToPreserve + totalCALLTypeStackArguments)
  )

  // we swapped with garbage stack which we no longer need since CALL has been executed, so POP
  op.push({ opcode: Opcode.POP, consumedBytes: undefined })
  // now that the success result is out of the way we can return memory to original state, the index and words are first on stack now!
  op = [...op, ...storeStackInMemory(numMemoryWordsToPreserve)]

  // lastly, POP all the original CALL params which were previously DUPed and modified appropriately.
  op = [
    ...op,
    ...new Array(1 + totalCALLTypeStackArguments - 1).fill({
      opcode: Opcode.POP,
      consumedBytes: undefined,
    }),
  ]

  return op
}

export const getEXTCODECOPYReplacement = (
  proxyAddress: Address,
  methodName: string
) => {
  const methodData: Buffer = abi.methodID(methodName, [])
  // stack params to EXTCODECOPY are:
  // [addr, destOffset, offset, length, …] (length = 4)
  // Technically the execution manager doesn't need to access the destOffset,
  // but it's just 32 bytes and would be more work to remove so we'll just pass it
  const numStackWordsToPass: number = 4

  // this is the number of memory words we'll be overwriting for the call.
  const numMemoryWordsToPreserve: number = 1 + 4 // 1 extra for methodId!

  let op: EVMBytecode = []
  // first, we push the memory we're gonna overwrite with calldata onto the stack, so that it may be parsed later.
  // to make sure there is not a collision between call and return data locations, we will store this AFTER (destOffset + length)
  op = [
    ...op,
    getDUPNOp(4), // DUP length
    getDUPNOp(3), // DUP destOffset
  ]
}
