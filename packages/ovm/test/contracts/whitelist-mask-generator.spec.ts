/* External Imports */
import { BigNumber, ZERO } from '@eth-optimism/core-utils'

/* Internal Imports */
import { EVMOpcode, Opcode } from '@eth-optimism/rollup-core'

const excludedOpCodes: EVMOpcode[] = [
  Opcode.ADDRESS,
  Opcode.BALANCE,
  Opcode.BLOCKHASH,
  Opcode.CALLCODE,
  Opcode.CALLER,
  Opcode.COINBASE,
  Opcode.CREATE,
  Opcode.CREATE2,
  Opcode.DELEGATECALL,
  Opcode.DIFFICULTY,
  Opcode.EXTCODESIZE,
  Opcode.EXTCODECOPY,
  Opcode.EXTCODEHASH,
  Opcode.GASLIMIT,
  Opcode.GASPRICE,
  Opcode.NUMBER,
  Opcode.ORIGIN,
  Opcode.SELFDESTRUCT,
  Opcode.SLOAD,
  Opcode.SSTORE,
  Opcode.STATICCALL,
  Opcode.TIMESTAMP,
]

const whitelistedOpcodes: EVMOpcode[] = Opcode.ALL_OP_CODES.filter(
  (x) => excludedOpCodes.indexOf(x) < 0
)

// TODO: Uncomment this to generate a new whitelist mask
describe('generates whitelist masks -- this is a util, not a test', () => {
  it('does the thing', () => {
    // Produces a hex number of 256 bits where each bit represents an
    // opcode, 0 - 255, which is set if whitelisted and unset otherwise.
    // Useful for the SafetyChecker  contract.
    console.log(
      `WHITELISTED OPCODES: ${whitelistedOpcodes.map((x) => x.name).join(',')}`
    )
    let whitelistMaskHex: string = whitelistedOpcodes
      .map((x) => new BigNumber(2).pow(new BigNumber(x.code)))
      .reduce((prev: BigNumber, cur: BigNumber) => prev.add(cur), ZERO)
      .toString('hex')
    if (whitelistMaskHex.length !== 64) {
      whitelistMaskHex =
        '0'.repeat(64 - whitelistMaskHex.length) + whitelistMaskHex
    }

    console.log(`mask: 0x${whitelistMaskHex}`)
  })
})
