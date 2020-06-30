import { TranspilationError } from '../../types/transpiler'
import {
  bufToHexString,
  getLogger,
  hexStrToBuf,
  Logger,
} from '@eth-optimism/core-utils'

export const BIG_ENOUGH_GAS_LIMIT: number = 100000000

const log: Logger = getLogger('transpiler-util')

/**
 * Util function to create TranspilationErrors.
 *
 * @param index The index of the byte in the input bytecode where the error originates.
 * @param error The TranspilationError error type.
 * @param message The error message.
 * @returns The constructed TranspilationError
 */
export const createError = (
  index: number,
  error: number,
  message: string
): TranspilationError => {
  return {
    index,
    error,
    message,
  }
}

export const stripAuxData = (
  bytecode: Buffer,
  buildJSON: any,
  isDeployedBytecode: boolean
  ): Buffer => {
  const auxDataObject = buildJSON.evm.legacyAssembly['.data']
  const auxData = auxDataObject['0']['.auxdata']
  let bytecodeWithoutAuxdata: Buffer
  const auxdataObjectKeys = Object.keys(auxDataObject)
  // deployed bytecode always has auxdata at the end, but constuction code may not.
  if (auxdataObjectKeys.length <= 1  || isDeployedBytecode) {
    log.debug(`Auxdata is at EOF, removing entirely from bytecode...`)
    const split = bufToHexString(bytecode).split(auxData)
    bytecodeWithoutAuxdata = hexStrToBuf(
      split[0]
    )
  } else {
    log.debug(`Auxdata is not at EOF, replacing it with 0s to preserve remaining data...`)
    const auxDataBuf: Buffer = hexStrToBuf(auxData)
    const auxDataPosition = bytecode.indexOf(auxDataBuf)
    log.debug(`buf: ${bufToHexString(auxDataBuf)}, position: ${auxDataPosition}, length: ${auxDataBuf.byteLength}`)
    bytecodeWithoutAuxdata = Buffer.from(bytecode)
    bytecodeWithoutAuxdata.fill(
      0,
      auxDataPosition,
      auxDataPosition + auxDataBuf.byteLength
    )
  }

  return bytecodeWithoutAuxdata
}