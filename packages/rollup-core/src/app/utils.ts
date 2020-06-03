/* External Imports */
import { bufToHexString } from '@eth-optimism/core-utils'

import { Contract, ContractFactory, Wallet } from 'ethers'

/* Internal Imports */
import { EVMBytecode, EVMOpcodeAndBytes, Opcode } from '../types'

/**
 * Creates an unsigned transaction and returns its calldata.
 *
 * @param contract The contract containing the function being invoked
 * @param functionName The function being invoked
 * @param args The arguments of the function call
 * @returns The unsigned transaction's calldata
 */
export const getUnsignedTransactionCalldata = (
  contract: Contract | ContractFactory,
  functionName: string,
  args: any[] = []
) => {
  return contract.interface.functions[functionName].encode(args)
}

/**
 * Takes EVMBytecode and serializes it into a single Buffer.
 *
 * @param bytecode The bytecode to serialize into a single Buffer.
 * @returns The resulting Buffer.
 */
export const bytecodeToBuffer = (bytecode: EVMBytecode): Buffer => {
  return Buffer.concat(
    bytecode.map((b) => {
      return b.consumedBytes !== undefined
        ? Buffer.concat([b.opcode.code, b.consumedBytes])
        : b.opcode.code
    })
  )
}

/**
 * Parses the provided Buffer into EVMBytecode.
 * Note: If the Buffer is not valid bytecode, this will throw.
 *
 * @param buffer The buffer in question.
 * @returns The parsed EVMBytecode.
 */
export const bufferToBytecode = (buffer: Buffer): EVMBytecode => {
  const bytecode: EVMBytecode = []

  for (let pc = 0; pc < buffer.length; pc++) {
    const opcode = Opcode.parseByNumber(buffer[pc])
    if (!opcode) {
      bytecode.push({
        opcode: {
          name: `UNRECOGNIZED (${bufToHexString(Buffer.from([buffer[pc]]))})`,
          code: Buffer.from([buffer[pc]]),
          programBytesConsumed: 0,
        },
        consumedBytes: undefined,
      })
      continue
    }
    const consumedBytes: Buffer =
      opcode.programBytesConsumed === 0
        ? undefined
        : buffer.slice(pc + 1, pc + 1 + opcode.programBytesConsumed)

    bytecode.push({
      opcode,
      consumedBytes,
    })

    pc += opcode.programBytesConsumed
  }
  return bytecode
}

/**
 * Gets the provided EVMBytecode as a printable string, where each line is an opcode and bytes.
 *
 * @param bytecode The EVMBytecode in question.
 * @returns The resulting string.
 */
export const formatBytecode = (bytecode: EVMBytecode): string => {
  return bytecode
    .map((x, index) => {
      let tagString: string = '(no tag)'
      if (!!x.tag) {
        tagString = `Metadata Tag: ${JSON.stringify(x.tag)}`
      }
      const pcAsString: string = padToLength(
        getPCOfEVMBytecodeIndex(index, bytecode),
        10
      )
      if (x.consumedBytes === undefined) {
        return `[PC ${pcAsString}] ${x.opcode.name} ${tagString}`
      }
      return `[PC ${pcAsString}] ${x.opcode.name}: ${bufToHexString(
        x.consumedBytes
      )} ${tagString}`
    })
    .join('\n')
}

const padToLength = (num: number, len: number): string => {
  const str = num.toString(16)
  return str.length >= len ? str : '0'.repeat(len - str.length) + str
}

/**
 * Gets the PC of the operation at a given index in some EVMBytecode.
 * In other words, it gives us the index of where a given element in some EVMBytecode would be in its raw Buffer form.
 *
 * @param indexOfEVMOpcodeAndBytes The index of an EVMOpcodeAndBytes element to find the PC of.
 * @param bytecode The EVMBytecode in question.
 * @returns The resulting index in raw bytes where the EVMOpcodeAndBytes begins.
 */
export const getPCOfEVMBytecodeIndex = (
  indexOfEVMOpcodeAndBytes: number,
  bytecode: EVMBytecode
): number => {
  let pc: number = 0
  for (let i = 0; i < indexOfEVMOpcodeAndBytes; i++) {
    const operation: EVMOpcodeAndBytes = bytecode[i]
    const totalBytesForOperation =
      operation.consumedBytes === undefined
        ? 1
        : 1 + operation.opcode.programBytesConsumed
    pc += totalBytesForOperation
  }
  return pc
}

export function getWallets(httpProvider) {
  const walletsToReturn = []
  for (let i = 0; i < 9; i++) {
    const privateKey = '0x' + ('5' + i).repeat(32)
    const nextWallet = new Wallet(privateKey, httpProvider)
    walletsToReturn[i] = nextWallet
  }
  return walletsToReturn
}

export async function deployContract(
  wallet,
  contractJSON,
  args,
  overrideOptions = {}
) {
  const factory = new ContractFactory(
    contractJSON.abi,
    contractJSON.bytecode || contractJSON.evm.bytecode,
    wallet
  )

  const contract = await factory.deploy(...args)
  await contract.deployed()
  return contract
}

/**
 * Gets the current number of seconds since the epoch.
 *
 * @returns The seconds since epoch.
 */
export function getCurrentTime(): number {
  return Math.round(new Date().getTime() / 1000)
}
