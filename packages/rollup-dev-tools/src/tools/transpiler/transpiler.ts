/* External Imports */
import {
  Opcode,
  EVMOpcodeAndBytes,
  EVMBytecode,
  bytecodeToBuffer,
  EVMOpcode,
  formatBytecode,
  bufferToBytecode,
  getPCOfEVMBytecodeIndex,
  OpcodeTagReason,
} from '@eth-optimism/rollup-core'
import {
  getLogger,
  bufToHexString,
  bufferUtils,
  add0x,
} from '@eth-optimism/core-utils'

import BigNum = require('bn.js')

/* Internal Imports */
import {
  OpcodeWhitelist,
  OpcodeReplacer,
  Transpiler,
  TranspilationResult,
  TranspilationError,
  TranspilationErrors,
  ErroredTranspilation,
  TaggedTranspilationResult,
  JumpReplacementResult,
} from '../../types/transpiler'
import { accountForJumps } from './jump-replacement'
import { isTaggedWithReason } from './helpers'

const log = getLogger('transpiler-impl')

export class TranspilerImpl implements Transpiler {
  constructor(
    private readonly opcodeWhitelist: OpcodeWhitelist,
    private readonly opcodeReplacer: OpcodeReplacer
  ) {
    if (!opcodeWhitelist) {
      throw Error('Opcode Whitelist is required for TranspilerImpl')
    }
    if (!opcodeReplacer) {
      throw Error('Opcode Replacer is required for TranspilerImpl')
    }
  }

  // This function transpiles initcode--that is, all bytecode fed to CREATE/CREATE2 other than the runtime-appended constructor inputs.
  // The Solidity compiler produces this initcode with the following pattern:
  // 1. constructor/deployment logic
  // 2. deployed bytecode
  // 3. constants accessed by the constructor
  // This pattern has been confirmed/explored extensively at ../../tests/transpiler/initcode-structure-check.spec.ts
  // The way we transpile this:
  // 1. Separate out 1, 2, and 3 above
  // 2. Tag the opcodes related to CODECOPY usage, the three types are 1. constants, 2. deployed bytecode to be returned, 3. inputs to the constructor.  Also, record the constants for case 1. so we know what they are.
  // 3. Transpile the deployed bytecode and correct the constant indices.
  // 4. Transpile the constructor/deployment logic.
  // 5. Reconstruct the transpiled constructor/deployment logic, deployed bytecode, and constants accessed by the constructor
  // 6. fix the remaining tagged CODECOPY indices which were detected in step 2.

  public transpile(
    bytecode: Buffer,
    deployedBytecode: Buffer,
    originalDeployedBytecodeSize: number = deployedBytecode.length
  ): TranspilationResult {
    const errors: TranspilationError[] = []

    log.debug(
      `transpiling raw full bytecode: ${bufToHexString(
        bytecode
      )} \nAnd original deployed raw bytecode: ${bufToHexString(
        deployedBytecode
      )}`
    )
    const startOfDeployedBytecode: number = bytecode.indexOf(deployedBytecode)
    if (startOfDeployedBytecode === -1) {
      const errMsg = `WARNING: Could not find deployed bytecode (${bufToHexString(
        deployedBytecode
      )}) within the original bytecode (${bufToHexString(bytecode)}).`
      log.debug(errMsg)
      errors.push(
        TranspilerImpl.createError(
          0,
          TranspilationErrors.MISSING_DEPLOYED_BYTECODE_ERROR,
          errMsg
        )
      )
    }

    // **** SEPARATE THE THREE SECTIONS OF THE INPUT BYTECODE ****
    // These sections are:
    // 1. constructor/deployment logic
    // 2. deployed bytecode
    // 3. constants accessed by the constructor (if any)

    const endOfDeployedBytecode: number =
      startOfDeployedBytecode + deployedBytecode.length
    let constantsUsedByConstructor: Buffer
    if (endOfDeployedBytecode < bytecode.length) {
      constantsUsedByConstructor = bytecode.slice(endOfDeployedBytecode)
      log.debug(
        `Detected constants being used by the constructor.  Together they are: \n${bufToHexString(
          constantsUsedByConstructor
        )}`
      )
    } else {
      log.debug('Did not detect any constants being used by the constructor.')
    }

    const originalDeployedEVMBytecode: EVMBytecode = bufferToBytecode(
      deployedBytecode
    )
    const originalConstructorInitLogic: EVMBytecode = bufferToBytecode(
      bytecode.slice(0, startOfDeployedBytecode)
    )

    log.debug(
      `original deployed evm bytecode is: ${formatBytecode(
        originalDeployedEVMBytecode
      )}`
    )

    // **** DETECT AND TAG THE CONSTANTS IN DEPLOYED BYTECODE AND TRANSPILE ****

    let taggedDeployedEVMBytecode: EVMBytecode
    taggedDeployedEVMBytecode = this.findAndTagConstants(
      originalDeployedEVMBytecode,
      deployedBytecode,
      errors
    )
    const deployedBytecodeTranspilationResult: TaggedTranspilationResult = this.transpileBytecodePreservingTags(
      taggedDeployedEVMBytecode
    )

    if (!deployedBytecodeTranspilationResult.succeeded) {
      errors.push(
        ...(deployedBytecodeTranspilationResult as ErroredTranspilation).errors
      )
      return {
        succeeded: false,
        errors,
      }
    }
    const transpiledDeployedBytecode: EVMBytecode =
      deployedBytecodeTranspilationResult.bytecodeWithTags

    log.debug(`Fixing the constant indices for transpiled deployed bytecode...`)
    log.debug(`errors are: ${JSON.stringify(errors)}`)
    // Note that fixTaggedConstantOffsets() scrubs all fixed tags, so we do not re-fix when we use finalTranspiledDeployedBytecode to reconstruct the returned initcode
    const finalTranspiledDeployedBytecode: EVMBytecode = this.fixTaggedConstantOffsets(
      transpiledDeployedBytecode as EVMBytecode,
      errors
    )

    // problem is after here?
    log.debug(
      `final transpiled deployed bytecode: \n${formatBytecode(
        finalTranspiledDeployedBytecode
      )}`
    )

    // **** DETECT AND TAG USES OF CODECOPY IN CONSTRUCTOR BYTECODE AND TRANSPILE ****

    let taggedOriginalConstructorInitLogic: EVMBytecode
    taggedOriginalConstructorInitLogic = this.findAndTagConstants(
      originalConstructorInitLogic,
      bytecode,
      errors
    )
    taggedOriginalConstructorInitLogic = this.findAndTagConstructorParamsLoader(
      originalConstructorInitLogic,
      errors,
      bytecode,
      originalDeployedBytecodeSize
    )
    taggedOriginalConstructorInitLogic = this.findAndTagDeployedBytecodeReturner(
      originalConstructorInitLogic
    )
    const constructorInitLogicTranspilationResult: TaggedTranspilationResult = this.transpileBytecodePreservingTags(
      taggedOriginalConstructorInitLogic
    )
    if (!constructorInitLogicTranspilationResult.succeeded) {
      errors.push(
        ...(constructorInitLogicTranspilationResult as ErroredTranspilation)
          .errors
      )
      return {
        succeeded: false,
        errors,
      }
    }
    const transpiledConstructorInitLogic: EVMBytecode =
      constructorInitLogicTranspilationResult.bytecodeWithTags

    // **** FIX THE TAGGED VALUES USED IN CONSTRUCTOR ****

    const transpiledInitLogicByteLength: number = bytecodeToBuffer(
      transpiledConstructorInitLogic
    ).byteLength

    const transpiledDeployedBytecodeByteLength: number = bytecodeToBuffer(
      transpiledDeployedBytecode
    ).byteLength

    const constantsUsedByConstructorLength: number = !constantsUsedByConstructor
      ? 0
      : constantsUsedByConstructor.byteLength

    for (const [index, op] of transpiledConstructorInitLogic.entries()) {
      if (
        isTaggedWithReason(op, [OpcodeTagReason.IS_CONSTRUCTOR_INPUTS_OFFSET])
      ) {
        // this should be the total length of the bytecode we're about to have generated!
        transpiledConstructorInitLogic[index].consumedBytes = new BigNum(
          transpiledInitLogicByteLength +
            transpiledDeployedBytecodeByteLength +
            constantsUsedByConstructorLength
        ).toBuffer('be', op.opcode.programBytesConsumed)
      }
      if (isTaggedWithReason(op, [OpcodeTagReason.IS_DEPLOY_CODE_LENGTH])) {
        transpiledConstructorInitLogic[index].consumedBytes = new BigNum(
          transpiledDeployedBytecodeByteLength
        ).toBuffer('be', op.opcode.programBytesConsumed)
      }
      if (isTaggedWithReason(op, [OpcodeTagReason.IS_DEPLOY_CODECOPY_OFFSET])) {
        transpiledConstructorInitLogic[index].consumedBytes = new BigNum(
          transpiledInitLogicByteLength
        ).toBuffer('be', op.opcode.programBytesConsumed)
      }
    }

    // **** FIX CONSTANTS IN THE INITCODE AND RETURN THE FINALIZED BYTECODE ****

    const finalTranspiledBytecode: EVMBytecode = this.fixTaggedConstantOffsets(
      [
        ...transpiledConstructorInitLogic,
        ...finalTranspiledDeployedBytecode,
        ...(!constantsUsedByConstructor
          ? []
          : bufferToBytecode(constantsUsedByConstructor)),
      ],
      errors
    )

    return {
      succeeded: true,
      bytecode: bytecodeToBuffer(finalTranspiledBytecode),
    }
  }

  // Fixes the tagged constants-loading offset in some transpiled bytecode.
  // Since we record the constant's value when we tag it, we can just search for the original value as a constant with indexOf() and set the index to that value.
  private fixTaggedConstantOffsets(
    taggedBytecode: EVMBytecode,
    errors
  ): EVMBytecode {
    log.debug(`tagged input: ${formatBytecode(taggedBytecode)}`)
    const inputAsBuf: Buffer = bytecodeToBuffer(taggedBytecode)
    const bytecodeToReturn: EVMBytecode = []
    for (const [index, op] of taggedBytecode.entries()) {
      bytecodeToReturn[index] = {
        opcode: taggedBytecode[index].opcode,
        consumedBytes: taggedBytecode[index].consumedBytes,
      }
      if (isTaggedWithReason(op, [OpcodeTagReason.IS_CONSTANT_OFFSET])) {
        const theConstant: Buffer = op.tag.metadata
        const newConstantOffset: number = inputAsBuf.indexOf(theConstant)
        if (newConstantOffset === -1) {
          errors.push(
            TranspilerImpl.createError(
              index,
              TranspilationErrors.MISSING_CONSTANT_ERROR,
              `Could not find CODECOPYed constant in transpiled deployed bytecode for PC 0x${index.toString(
                16
              )}.  We originally recorded the constant as ${bufToHexString(
                theConstant
              )}, but it does not exist in the post-transpiled ${bufToHexString(
                inputAsBuf
              )}`
            )
          )
        }
        const newConstantOffsetBuf: Buffer = new BigNum(
          newConstantOffset
        ).toBuffer('be', op.opcode.programBytesConsumed)
        log.debug(
          `fixing CODECOPY(constant) at PC 0x${getPCOfEVMBytecodeIndex(
            index,
            taggedBytecode
          ).toString(16)}.  Setting new index to 0x${bufToHexString(
            newConstantOffsetBuf
          )}`
        )
        bytecodeToReturn[index].consumedBytes = newConstantOffsetBuf
      }
    }
    return bytecodeToReturn
  }

  // Finds and tags the PUSHN's which are detected to be associated with CODECOPYing deployed bytecode which is returned during CREATE/CREATE2.
  // See https://github.com/ethereum-optimism/optimistic-rollup/wiki/CODECOPYs for more details.
  private findAndTagDeployedBytecodeReturner(
    bytecode: EVMBytecode
  ): EVMBytecode {
    for (let index = 0; index < bytecode.length - 6; index++) {
      const op: EVMOpcodeAndBytes = bytecode[index]
      // Tags based on the pattern used for deploying non-library contracts:
      // PUSH2 // codecopy's and RETURN's length
      // DUP1 // DUPed to use twice, for RETURN and CODECOPY both
      // PUSH2 // codecopy's offset
      // PUSH1 codecopy's destOffset
      // CODECOPY // copy
      // PUSH1 0 // RETURN offset
      // RETURN // uses above RETURN offset and DUP'ed length above
      if (
        Opcode.isPUSHOpcode(op.opcode) &&
        Opcode.isPUSHOpcode(bytecode[index + 2].opcode) &&
        bytecode[index + 4].opcode === Opcode.CODECOPY &&
        bytecode[index + 6].opcode === Opcode.RETURN
      ) {
        log.debug(
          `detected a NON-LIBRARY [CODECOPY(deployed bytecode)... RETURN] (CREATE/2 deployment logic) pattern starting at PC: 0x${getPCOfEVMBytecodeIndex(
            index,
            bytecode
          ).toString(16)}. Tagging the offset and size...`
        )
        bytecode[index] = {
          opcode: op.opcode,
          consumedBytes: op.consumedBytes,
          tag: {
            padPUSH: true,
            reasonTagged: OpcodeTagReason.IS_DEPLOY_CODE_LENGTH,
            metadata: undefined,
          },
        }
        bytecode[index + 2] = {
          opcode: bytecode[index + 2].opcode,
          consumedBytes: bytecode[index + 2].consumedBytes,
          tag: {
            padPUSH: true,
            reasonTagged: OpcodeTagReason.IS_DEPLOY_CODECOPY_OFFSET,
            metadata: undefined,
          },
        }
      }
      // Tags based on the pattern used for deploying library contracts:
      // PUSH2 // deployed bytecode length
      // PUSH2 // deployed bytecode start
      // PUSH1: // destoffset of code to copy
      // DUP3
      // DUP3
      // DUP3
      // CODECOPY
      else if (
        Opcode.isPUSHOpcode(op.opcode) &&
        Opcode.isPUSHOpcode(bytecode[index + 1].opcode) &&
        Opcode.isPUSHOpcode(bytecode[index + 2].opcode) &&
        bytecode[index + 6].opcode === Opcode.CODECOPY
      ) {
        log.debug(
          `detected a LIBRARY [CODECOPY(deployed bytecode)... RETURN] (library deployment logic) pattern starting at PC: 0x${getPCOfEVMBytecodeIndex(
            index,
            bytecode
          ).toString(16)}. Tagging the offset and size...`
        )
        bytecode[index] = {
          opcode: op.opcode,
          consumedBytes: op.consumedBytes,
          tag: {
            padPUSH: true,
            reasonTagged: OpcodeTagReason.IS_DEPLOY_CODE_LENGTH,
            metadata: undefined,
          },
        }
        bytecode[index + 1] = {
          opcode: bytecode[index + 1].opcode,
          consumedBytes: bytecode[index + 1].consumedBytes,
          tag: {
            padPUSH: true,
            reasonTagged: OpcodeTagReason.IS_DEPLOY_CODECOPY_OFFSET,
            metadata: undefined,
          },
        }
      }
    }
    return bytecode
  }

  // Finds and tags the PUSHN's which are detected to be associated with CODECOPYing constructor params during CREATE/CREATE2.
  // Tags based on the pattern:
  // PUSH2 // should be initcode.length + deployedbytecode.length
  // CODESIZE
  // SUB // subtract however big the code is from the amount pushed above to get the length of constructor input
  // DUP1
  // PUSH2 // should also be initcode.length + deployedbytecode.length
  // DUP4
  // CODECOPY
  // See https://github.com/ethereum-optimism/optimistic-rollup/wiki/CODECOPYs for more details.

  /* Inputs:
   * bytecode: EVMBytcode  - the subset of bytecode to tag.
   * fullBytecodeBuf: Buffer - the full bytes of the code in which the subset will run.  Used to grab constructor constants which come AFTER deployed bytecode, while only tagging the constructor logic.
   * sizeIncreaseFromPreviousPadding: any increase in length which the bytecode has experienced since being output from solc-js
   * Outputs:
   * EVMBytecode - the Bytecode, but with the constructor params loader PUSHes tagged.
   */
  private findAndTagConstructorParamsLoader(
    bytecode: EVMBytecode,
    errors,
    fullBytecodeBuf: Buffer,
    originalDeployedBytecodeSize: number = fullBytecodeBuf.byteLength
  ): EVMBytecode {
    for (let index = 0; index < bytecode.length - 6; index++) {
      const op: EVMOpcodeAndBytes = bytecode[index]
      if (
        Opcode.isPUSHOpcode(op.opcode) &&
        bytecode[index + 1].opcode === Opcode.CODESIZE &&
        bytecode[index + 2].opcode === Opcode.SUB &&
        Opcode.isPUSHOpcode(bytecode[index + 4].opcode) &&
        bytecode[index + 6].opcode === Opcode.CODECOPY
      ) {
        const pushedOffset: number = new BigNum(op.consumedBytes).toNumber()
        if (pushedOffset !== originalDeployedBytecodeSize) {
          errors.push(
            TranspilerImpl.createError(
              index,
              TranspilationErrors.DETECTED_CONSTANT_OOB,
              `thought we were in a CODECOPY(constructor params), but wrong length...at PC: 0x${getPCOfEVMBytecodeIndex(
                index,
                bytecode
              ).toString(
                16
              )}.  PUSH of offset which we thought was the total initcode length was 0x${pushedOffset.toString(
                16
              )}, but length of original bytecode was specified or detected to be 0x${originalDeployedBytecodeSize}`
            )
          )
        }
        log.debug(
          `Successfully detected a CODECOPY(constructor params) pattern starting at PC: 0x${getPCOfEVMBytecodeIndex(
            index,
            bytecode
          ).toString(16)}.`
        )
        bytecode[index] = {
          opcode: op.opcode,
          consumedBytes: op.consumedBytes,
          tag: {
            padPUSH: true,
            reasonTagged: OpcodeTagReason.IS_CONSTRUCTOR_INPUTS_OFFSET,
            metadata: undefined,
          },
        }
        bytecode[index + 4] = {
          opcode: bytecode[index + 4].opcode,
          consumedBytes: bytecode[index + 4].consumedBytes,
          tag: {
            padPUSH: true,
            reasonTagged: OpcodeTagReason.IS_CONSTRUCTOR_INPUTS_OFFSET,
            metadata: undefined,
          },
        }
      }
    }
    return bytecode
  }

  // Finds and tags the PUSHN's which are detected to be associated with CODECOPYing constants.
  // Tags based on the pattern:
  //   ...
  // PUSH2 // offset of constant in bytecode
  // PUSH1 // length of constant
  // SWAP2 // where to put it into memory
  // CODECOPY
  // It also copies the constants into the tag so that their new position can be recovered later.
  // See https://github.com/ethereum-optimism/optimistic-rollup/wiki/CODECOPYs for more details.
  public findAndTagConstants(
    bytecode: EVMBytecode,
    fullRawBytecode: Buffer,
    errors
  ): EVMBytecode {
    const taggedBytecode: EVMBytecode = bytecode as EVMBytecode
    for (let index = 0; index < bytecode.length - 3; index++) {
      // this pattern is 3 long, so stop 2 early
      const op: EVMOpcodeAndBytes = bytecode[index]

      // log.debug(`cur index tagging constants: 0x${getPCOfEVMBytecodeIndex(index, bytecode).toString(16)}`)
      // log.debug(`at this index we see the following opcodes: \n${formatBytecode(bytecode.slice(index, index + 10))}`)
      if (
        Opcode.isPUSHOpcode(op.opcode) &&
        Opcode.isPUSHOpcode(bytecode[index + 1].opcode) &&
        bytecode[index + 3].opcode === Opcode.CODECOPY
      ) {
        const offsetForCODECOPY: number = new BigNum(
          op.consumedBytes
        ).toNumber()
        const lengthforCODECOPY: number = new BigNum(
          bytecode[index + 1].consumedBytes
        ).toNumber()
        const constantStart: number = offsetForCODECOPY
        const constantEnd: number = constantStart + lengthforCODECOPY
        if (constantEnd > fullRawBytecode.byteLength) {
          errors.push(
            TranspilerImpl.createError(
              index,
              TranspilationErrors.DETECTED_CONSTANT_OOB,
              `Thought we detected a CODECOP(a CODECOPY(constant) pattern at starting at PC: 0x${getPCOfEVMBytecodeIndex(
                index,
                bytecode
              ).toString(
                16
              )}, but it is out of bounds (not part of the bytecode))`
            )
          )
        }
        const theConstant: Buffer = fullRawBytecode.slice(
          constantStart,
          constantEnd
        )
        log.debug(
          `detected a CODECOPY(constant) pattern at starting at PC: 0x${getPCOfEVMBytecodeIndex(
            index,
            bytecode
          ).toString(16)}.  Its val: ${bufToHexString(theConstant)}`
        )
        taggedBytecode[index] = {
          opcode: op.opcode,
          consumedBytes: op.consumedBytes,
          tag: {
            padPUSH: true,
            reasonTagged: OpcodeTagReason.IS_CONSTANT_OFFSET,
            metadata: theConstant,
          },
        }
      }
    }
    return taggedBytecode
  }

  // This function transpiles "deployed bytecode"-type bytecode, operating on potentially tagged EVMBytecode (==EVMOpcodeAndBytes[]).
  // It preserves all .tags values of the EVMOpcodeAndBytes UNLESS:
  // 1. The opcode is replaced by the replacer. (aka getSubstituedFunctionFor() does not just return the input
  // 2. The EVMOpcodeAndBytes is a JUMP/JUMPI/JUMPDEST (aka affected by the JUMP table)
  private transpileBytecodePreservingTags(
    bytecode: EVMBytecode
  ): TaggedTranspilationResult {
    let transpiledBytecode: EVMBytecode = []
    const errors: TranspilationError[] = []
    const jumpdestIndexesBefore: number[] = []
    let lastOpcode: EVMOpcode
    let insideUnreachableCode: boolean = false
    const replacedOpcodes: Set<EVMOpcode> = new Set<EVMOpcode>()

    const [lastOpcodeAndConsumedBytes] = bytecode.slice(-1)
    if (
      Opcode.isPUSHOpcode(lastOpcodeAndConsumedBytes.opcode) &&
      lastOpcodeAndConsumedBytes.consumedBytes.byteLength <
        lastOpcodeAndConsumedBytes.opcode.programBytesConsumed
    ) {
      // todo: handle with warnings[] separate from errors[]?
      const message: string = `Final input opcode: ${
        lastOpcodeAndConsumedBytes.opcode.name
      } consumes ${
        lastOpcodeAndConsumedBytes.opcode.programBytesConsumed
      }, but only has 0x${bufToHexString(
        lastOpcodeAndConsumedBytes.consumedBytes
      )} following it.  Padding with zeros under the assumption that this arises from a constant at EOF...`
      log.debug(message)
      lastOpcodeAndConsumedBytes.consumedBytes = bufferUtils.padRight(
        lastOpcodeAndConsumedBytes.consumedBytes,
        lastOpcodeAndConsumedBytes.opcode.programBytesConsumed
      )
    }

    const bytecodeBuf: Buffer = bytecodeToBuffer(bytecode)
    // todo remove once confirmed with Kevin?
    let seenJump: boolean = false
    // track the index in EVMBytecode we are on so that we can preserve metadata when we append it
    // incrementing at the beginning of the loop so start at -1
    let indexOfOpcodeAndBytes: number = -1
    for (let pc = 0; pc < bytecodeBuf.length; pc++) {
      indexOfOpcodeAndBytes += 1
      const currentTaggedOpcodeAndBytes: EVMOpcodeAndBytes =
        bytecode[indexOfOpcodeAndBytes]

      let opcode = Opcode.parseByNumber(bytecodeBuf[pc])
      // If we are inside unreachable code, and see a JUMPDEST, the code is now reachable
      if (insideUnreachableCode && seenJump && opcode === Opcode.JUMPDEST) {
        insideUnreachableCode = false
      }
      if (!insideUnreachableCode) {
        if (
          !TranspilerImpl.validOpcode(
            opcode,
            pc,
            bytecodeBuf[pc],
            lastOpcode,
            errors
          )
        ) {
          log.debug(
            `Deteced invalid opcode in reachable code: ${opcode}. at PC: 0x${pc.toString(
              16
            )} Skipping inclusion in transpilation output...`
          )
          // skip, do not push anything to the transpilation output
          lastOpcode = undefined
          continue
        }
        lastOpcode = opcode
        seenJump = seenJump || Opcode.JUMP_OP_CODES.includes(opcode)
        insideUnreachableCode = Opcode.HALTING_OP_CODES.includes(opcode)

        if (opcode === Opcode.JUMPDEST) {
          jumpdestIndexesBefore.push(pc)
        }
        if (!this.opcodeWhitelisted(opcode, pc, errors)) {
          pc += opcode.programBytesConsumed
          continue
        }
      }
      if (insideUnreachableCode && !opcode) {
        const unreachableCode: Buffer = bytecodeBuf.slice(pc, pc + 1)
        opcode = {
          name: `UNREACHABLE (${bufToHexString(unreachableCode)})`,
          code: unreachableCode,
          programBytesConsumed: 0,
        }
      }

      const tag = currentTaggedOpcodeAndBytes.tag
      const opcodeAndBytes: EVMOpcodeAndBytes = {
        opcode,
        consumedBytes: !opcode.programBytesConsumed
          ? undefined
          : bytecodeBuf.slice(pc + 1, pc + 1 + opcode.programBytesConsumed),
        tag,
      }

      if (!!tag && tag.padPUSH) {
        opcodeAndBytes.consumedBytes = Buffer.concat([
          Buffer.alloc(1),
          opcodeAndBytes.consumedBytes,
        ])
        // will break if we ever tagged a push32 because push33 doesn't exist.  However we shouldn't be tagging any such val.
        opcodeAndBytes.opcode = Opcode.parseByNumber(
          Opcode.getCodeNumber(opcodeAndBytes.opcode) + 1
        )
      }
      let transpiledBytecodeReplacement: EVMBytecode
      if (
        insideUnreachableCode ||
        !this.opcodeReplacer.shouldSubstituteOpcodeForFunction(
          opcodeAndBytes.opcode
        )
      ) {
        transpiledBytecodeReplacement = [opcodeAndBytes]
      } else {
        // record that we will need to add this opcode to the replacement table
        replacedOpcodes.add(opcodeAndBytes.opcode)
        // jump to the footer where the logic of the replacement will be executed
        transpiledBytecodeReplacement = this.opcodeReplacer.getJUMPToOpcodeFunction(
          opcodeAndBytes.opcode
        )
      }

      transpiledBytecode.push(...transpiledBytecodeReplacement)
      pc += opcode.programBytesConsumed
    }

    log.debug(
      `Bytecode after replacement before JUMP logic: \n${formatBytecode(
        transpiledBytecode
      )}`
    )

    const res: JumpReplacementResult = accountForJumps(
      transpiledBytecode,
      jumpdestIndexesBefore
    )
    // TODO make sure accountForJumps STOPs after, should do
    errors.push(...(res.errors || []))
    const bytecodeWithTranspiledJumpsPopulated = res.bytecode

    log.debug(
      `Bytecode after replacement and fixed existing JUMP logic: \n${formatBytecode(
        bytecodeWithTranspiledJumpsPopulated
      )}`
    )

    const opcodeReplacementFooter: EVMBytecode = this.opcodeReplacer.getOpcodeFunctionTable(
      replacedOpcodes
    )
    log.debug(
      `Inserting opcode replacement footer: ${formatBytecode(
        opcodeReplacementFooter
      )}`
    )
    transpiledBytecode = [
      ...bytecodeWithTranspiledJumpsPopulated,
      ...opcodeReplacementFooter,
    ]

    transpiledBytecode = this.opcodeReplacer.populateOpcodeFunctionJUMPs(
      transpiledBytecode
    )

    if (!!errors.length) {
      return {
        succeeded: false,
        errors,
      }
    }
    return {
      succeeded: true,
      bytecodeWithTags: transpiledBytecode,
    }
  }

  public transpileRawBytecode(bytecodeBuf: Buffer): TranspilationResult {
    const rawBytecodeTyped: EVMBytecode = bufferToBytecode(bytecodeBuf)
    const transpilationResult: TaggedTranspilationResult = this.transpileBytecodePreservingTags(
      rawBytecodeTyped
    )
    if (!transpilationResult.succeeded) {
      return {
        succeeded: false,
        errors: transpilationResult.errors,
      }
    }
    log.debug(
      `successfully executed transpileRawBytecode, got result: \n${formatBytecode(
        transpilationResult.bytecodeWithTags
      )}`
    )
    return {
      succeeded: true,
      bytecode: bytecodeToBuffer(transpilationResult.bytecodeWithTags),
    }
  }

  /**
   * Returns whether or not the provided EVMOpcode is valid (not undefined).
   * If it is not, it creates a new TranpilationError and appends it to the provided list.
   *
   * @param opcode The opcode in question.
   * @param pc The current program counter value.
   * @param code The code (decimal) of the opcode in question .
   * @param lastOpcode The last Opcode seen before this one.
   * @param errors The cumulative errors list.
   * @returns True if valid, False otherwise.
   */
  private static validOpcode(
    opcode: EVMOpcode,
    pc: number,
    code: number,
    lastOpcode: EVMOpcode,
    errors: TranspilationError[]
  ): boolean {
    if (!opcode) {
      let messageExtension: string = ''
      if (!!lastOpcode && !!lastOpcode.programBytesConsumed) {
        messageExtension = ` Was ${lastOpcode.name} at index ${pc -
          lastOpcode.programBytesConsumed} provided exactly ${
          lastOpcode.programBytesConsumed
        } bytes as expected?`
      }
      const message: string = `Cannot find opcode for: ${add0x(
        code.toString(16)
      )}.${messageExtension}`
      log.debug(message)
      errors.push(
        TranspilerImpl.createError(
          pc,
          TranspilationErrors.UNSUPPORTED_OPCODE,
          message
        )
      )
      return false
    }
    return true
  }

  /**
   * Returns whether or not the provided EVMOpcode is whitelisted.
   * If it is not, it creates a new TranpilationError and appends it to the provided list.
   *
   * @param opcode The opcode in question.
   * @param pc The current program counter value.
   * @param errors The cumulative errors list.
   * @returns True if whitelisted, False otherwise.
   */
  private opcodeWhitelisted(
    opcode: EVMOpcode,
    pc: number,
    errors: TranspilationError[]
  ): boolean {
    if (!this.opcodeWhitelist.isOpcodeWhitelisted(opcode)) {
      const message: string = `Opcode [${opcode.name}] is not on the whitelist.`
      log.debug(message)
      errors.push(
        TranspilerImpl.createError(
          pc,
          TranspilationErrors.OPCODE_NOT_WHITELISTED,
          message
        )
      )
      return false
    }
    return true
  }

  /**
   * Util function to create TranspilationErrors.
   *
   * @param index The index of the byte in the input bytecode where the error originates.
   * @param error The TranspilationErrors error type.
   * @param message The error message.
   * @returns The constructed TranspilationError
   */
  private static createError(
    index: number,
    error: number,
    message: string
  ): TranspilationError {
    return {
      index,
      error,
      message,
    }
  }
}
