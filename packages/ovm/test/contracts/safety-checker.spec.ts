/* Internal Imports */
import '../setup'

/* External Imports */
import { getLogger, add0x, remove0x } from '@eth-optimism/core-utils'
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  DEFAULT_UNSAFE_OPCODES,
  EVMOpcode,
  Opcode,
} from '@eth-optimism/rollup-core'
import { SafetyCheckerContractDefinition as SafetyChecker } from '@eth-optimism/rollup-contracts'

import { Contract } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Logging */
const log = getLogger('safety-checker', true)

const executionManagerAddress = add0x('12'.repeat(20)) // Test Execution Manager address 0x121...212
const haltingOpcodes: EVMOpcode[] = Opcode.HALTING_OP_CODES
const haltingOpcodesNoJump: EVMOpcode[] = haltingOpcodes.filter(
  (x) => x.name !== 'JUMP'
)
const jumps: EVMOpcode[] = [Opcode.JUMP, Opcode.JUMPI]
const whitelistedNotHaltingOrCALL: EVMOpcode[] = Opcode.ALL_OP_CODES.filter(
  (x) =>
    DEFAULT_UNSAFE_OPCODES.indexOf(x) < 0 &&
    haltingOpcodes.indexOf(x) < 0 &&
    x.name !== 'CALL'
)

describe('Safety Checker', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let safetyChecker: Contract

  /* Deploy a new whitelist contract before each test */
  beforeEach(async () => {
    safetyChecker = await deployContract(
      wallet,
      SafetyChecker,
      [DEFAULT_OPCODE_WHITELIST_MASK, executionManagerAddress],
      { gasLimit: 6700000 }
    )
  })

  describe('isBytecodeSafe()', async () => {
    describe('Empty case', () => {
      it('should work for empty case', async () => {
        const res: boolean = await safetyChecker.isBytecodeSafe([])
        res.should.eq(true, `empty bytecode should be whitelisted!`)
      })
    })

    describe('Single op-code cases', async () => {
      it('should correctly classify non-whitelisted', async () => {
        for (const opcode of DEFAULT_UNSAFE_OPCODES) {
          const res: boolean = await safetyChecker.isBytecodeSafe(
            `0x${opcode.code.toString('hex')}`
          )
          res.should.eq(
            false,
            `Opcode ${opcode.name} whitelisted by contract when it shouldn't be!`
          )
        }
      })

      it('should correctly classify whitelisted', async () => {
        for (const opcode of whitelistedNotHaltingOrCALL) {
          if (!opcode.name.startsWith('PUSH')) {
            const res: boolean = await safetyChecker.isBytecodeSafe(
              `0x${opcode.code.toString('hex')}`
            )
            res.should.eq(
              true,
              `Opcode ${opcode.name} not whitelisted by contract when it should be!`
            )
          }
        }
      })
    })

    describe('PUSH cases', async () => {
      it('skips at least specified number of bytes for PUSH cases', async () => {
        const invalidOpcode: string = DEFAULT_UNSAFE_OPCODES[0].code.toString(
          'hex'
        )
        const push1Code: number = parseInt(
          Opcode.PUSH1.code.toString('hex'),
          16
        )
        for (let i = 1; i < 33; i++) {
          const bytecode: string = `0x${Buffer.of(push1Code + i - 1).toString(
            'hex'
          )}${invalidOpcode.repeat(i)}`
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            true,
            `PUSH${i} failed by not skipping ${i} bytes of bytecode!`
          )
        }
      })

      it('skips at most specified number of bytes for PUSH cases', async () => {
        const invalidOpcode: string = DEFAULT_UNSAFE_OPCODES[0].code.toString(
          'hex'
        )
        const push1Code: number = parseInt(
          Opcode.PUSH1.code.toString('hex'),
          16
        )
        for (let i = 1; i < 33; i++) {
          const bytecode: string = `0x${Buffer.of(push1Code + i - 1).toString(
            'hex'
          )}${invalidOpcode.repeat(i + 1)}`
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            false,
            `PUSH${i} succeeded, skipping ${i +
              1} bytes of bytecode when it should have skipped ${i} bytes!`
          )
        }
      })
    })

    describe('multiple opcode cases', async () => {
      it('works for whitelisted, non-halting codes', async () => {
        let bytecode: string = '0x'
        const invalidOpcode: string = DEFAULT_UNSAFE_OPCODES[0].code.toString(
          'hex'
        )

        for (const opcode of whitelistedNotHaltingOrCALL) {
          bytecode += `${opcode.code.toString('hex')}${invalidOpcode.repeat(
            opcode.programBytesConsumed
          )}`
        }

        const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
        res.should.eq(true, `Bytecode of all whitelisted (non-halting) failed!`)
      })

      it('fails for non-halting whitelisted codes with one not on whitelist at the end', async () => {
        let bytecode: string = '0x'
        const invalidOpcode: string = DEFAULT_UNSAFE_OPCODES[0].code.toString(
          'hex'
        )

        for (const opcode of whitelistedNotHaltingOrCALL) {
          bytecode += `${opcode.code.toString('hex')}${invalidOpcode.repeat(
            opcode.programBytesConsumed
          )}`
        }
        for (const opcode of DEFAULT_UNSAFE_OPCODES) {
          const res: boolean = await safetyChecker.isBytecodeSafe(
            bytecode + opcode.code.toString('hex')
          )
          res.should.eq(
            false,
            `Bytecode of all whitelisted (non-halting) + ${opcode.name} passed when it should have failed!`
          )
        }
      }).timeout(30_000)
    })

    describe('handles unreachable code', async () => {
      it(`skips unreachable bytecode after a halting opcode`, async () => {
        for (const haltingOp of haltingOpcodes) {
          let bytecode: string = '0x'
          bytecode += haltingOp.code.toString('hex')
          for (const opcode of DEFAULT_UNSAFE_OPCODES) {
            bytecode += opcode.code.toString('hex')
          }
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            true,
            `Bytecode containing invalid opcodes in unreachable code after a ${haltingOp.name} failed!`
          )
        }
      })
      it('skips bytecode after an unreachable JUMPDEST', async () => {
        for (const haltingOp of haltingOpcodesNoJump) {
          let bytecode: string = '0x'
          bytecode += haltingOp.code.toString('hex')
          bytecode += Opcode.JUMPDEST.code.toString('hex')
          for (const opcode of DEFAULT_UNSAFE_OPCODES) {
            bytecode += opcode.code.toString('hex')
          }
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            true,
            `Bytecode containing invalid opcodes after unreachable JUMPDEST (after a ${haltingOp.name}) failed!`
          )
        }
      })

      it('parses opcodes after a reachable JUMPDEST', async () => {
        for (const haltingOp of haltingOpcodesNoJump) {
          for (const jump of jumps) {
            let bytecode: string = '0x'
            bytecode += jump.code.toString('hex')
            bytecode += Opcode.JUMPDEST.code.toString('hex') // JUMPDEST here so that the haltingOp is reachable
            bytecode += haltingOp.code.toString('hex')
            bytecode += Opcode.JUMPDEST.code.toString('hex')
            for (const opcode of DEFAULT_UNSAFE_OPCODES) {
              bytecode += opcode.code.toString('hex')
            }
            const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
            res.should.eq(
              false,
              `Bytecode containing invalid opcodes after a JUMPDEST preceded by a ${haltingOp.name} and reachable by ${jump.name} should have failed!`
            )
          }
        }
      })

      it('parses opcodes after first JUMP and JUMPDEST', async () => {
        let bytecode: string = '0x'
        bytecode += Opcode.JUMP.code.toString('hex')
        bytecode += Opcode.JUMPDEST.code.toString('hex')
        for (const opcode of DEFAULT_UNSAFE_OPCODES) {
          bytecode += opcode.code.toString('hex')
        }
        const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
        res.should.eq(
          false,
          `Bytecode containing invalid opcodes after reachble JUMPDEST should have failed!`
        )
      })

      it('parses opcodes after JUMPI', async () => {
        let bytecode: string = '0x'
        bytecode += Opcode.JUMPI.code.toString('hex')
        for (const opcode of DEFAULT_UNSAFE_OPCODES) {
          bytecode += opcode.code.toString('hex')
        }
        const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
        res.should.eq(
          false,
          `Bytecode containing invalid opcodes after JUMPI should have failed!`
        )
      })

      it('should correctly handle alternating reachable/uncreachable code ending in reachable, valid code', async () => {
        for (const haltingOp of haltingOpcodesNoJump) {
          for (const jump of jumps) {
            let bytecode: string = '0x'
            bytecode += jump.code.toString('hex')
            // JUMPDEST here so that the haltingOp is reachable
            bytecode += Opcode.JUMPDEST.code.toString('hex')
            for (let i = 0; i < 3; i++) {
              bytecode += haltingOp.code.toString('hex')
              // Unreachable, invalid code
              for (const opcode of DEFAULT_UNSAFE_OPCODES) {
                bytecode += opcode.code.toString('hex')
              }
              bytecode += Opcode.JUMPDEST.code.toString('hex')
              // Reachable, valid code
              for (const opcode of whitelistedNotHaltingOrCALL) {
                bytecode += opcode.code.toString('hex')
              }
            }
            const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
            res.should.eq(
              true,
              `Long bytecode containing alternating valid reachable and invalid unreachable code failed!`
            )
          }
        }
      }).timeout(30_000)

      it('should correctly handle alternating reachable/uncreachable code ending in reachable, invalid code', async () => {
        for (const haltingOp of haltingOpcodesNoJump) {
          for (const jump of jumps) {
            let bytecode: string = '0x'
            bytecode += jump.code.toString('hex')
            // JUMPDEST here so that the haltingOp is reachable
            bytecode += Opcode.JUMPDEST.code.toString('hex')
            for (let i = 0; i < 3; i++) {
              bytecode += haltingOp.code.toString('hex')
              // Unreachable, invalid code
              for (const opcode of DEFAULT_UNSAFE_OPCODES) {
                bytecode += opcode.code.toString('hex')
              }
              bytecode += Opcode.JUMPDEST.code.toString('hex')
              // Reachable, valid code
              for (const opcode of whitelistedNotHaltingOrCALL) {
                bytecode += opcode.code.toString('hex')
              }
            }
            bytecode += DEFAULT_UNSAFE_OPCODES[0].code.toString('hex')
            const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
            res.should.eq(
              false,
              `Long bytecode ending in reachable, invalid code should have failed!`
            )
          }
        }
      }).timeout(30_000)
    })
    describe('handles CALLs', async () => {
      it(`accepts valid call, PUSHing gas`, async () => {
        const invalidOpcode: string = DEFAULT_UNSAFE_OPCODES[0].code.toString(
          'hex'
        )
        const push1Code: number = parseInt(
          Opcode.PUSH1.code.toString('hex'),
          16
        )
        // test for PUSH1...PUSH32
        for (let i = 1; i <= 32; i++) {
          let bytecode: string = '0x'
          // set value
          bytecode += Opcode.PUSH1.code.toString('hex')
          bytecode += '00' //PUSH1 0x00
          // set address
          bytecode += Opcode.PUSH20.code.toString('hex')
          bytecode += remove0x(executionManagerAddress) //PUSH20 Execution Manager address
          // set gas
          bytecode += Buffer.of(push1Code + i - 1).toString('hex')
          bytecode += invalidOpcode.repeat(i)
          // CALL
          bytecode += Opcode.CALL.code.toString('hex')
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            true,
            `Bytecode containing valid CALL using PUSH${i} to set gas failed!`
          )
        }
      })
      it(`accepts valid call, DUPing gas`, async () => {
        const invalidOpcode: string = DEFAULT_UNSAFE_OPCODES[0].code.toString(
          'hex'
        )
        const dup1Code: number = parseInt(Opcode.DUP1.code.toString('hex'), 16)
        // test for DUP1...DUP16
        for (let i = 1; i <= 16; i++) {
          let bytecode: string = '0x'
          // set value
          bytecode += Opcode.PUSH1.code.toString('hex')
          bytecode += '00' //PUSH1 0x00
          // set address
          bytecode += Opcode.PUSH20.code.toString('hex')
          bytecode += remove0x(executionManagerAddress) //PUSH20 Execution Manager address
          // set gas
          bytecode += Buffer.of(dup1Code + i - 1).toString('hex')
          // CALL
          bytecode += Opcode.CALL.code.toString('hex')
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            true,
            `Bytecode containing valid CALL using DUP${i} to set gas failed!`
          )
        }
      })
      it(`rejects invalid CALLs using opcodes other than PUSH or DUP to set gas`, async () => {
        const invalidGasSetters: EVMOpcode[] = whitelistedNotHaltingOrCALL.filter(
          (x) => !x.name.startsWith('PUSH') && !x.name.startsWith('DUP')
        )
        log.debug(`Invalid Gas Setters ${invalidGasSetters.map((x) => x.name)}`)
        // test for whitelisted, non-halting opcodes (excluding PUSHes or DUPs)
        for (const opcode of invalidGasSetters) {
          let bytecode: string = '0x'
          // set value
          bytecode += Opcode.PUSH1.code.toString('hex')
          bytecode += '00' //PUSH1 0x00
          // set address
          bytecode += Opcode.PUSH20.code.toString('hex')
          bytecode += remove0x(executionManagerAddress) //PUSH20 Execution Manager address
          // set gas with invalid opcode
          bytecode += opcode.code.toString('hex')
          // CALL
          bytecode += Opcode.CALL.code.toString('hex')
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            false,
            `Bytecode containing invalid CALL using ${opcode.name} to set gas should have failed!`
          )
        }
      })
      it(`rejects invalid CALLs using opcodes other than PUSH1 to set value`, async () => {
        const invalidValueSetters: EVMOpcode[] = whitelistedNotHaltingOrCALL.filter(
          (x) => x.name !== 'PUSH1'
        )
        log.debug(
          `Invalid Value Setters ${invalidValueSetters.map((x) => x.name)}`
        )
        // test for whitelisted, non-halting opcodes (excluding PUSH1)
        for (const opcode of invalidValueSetters) {
          let bytecode: string = '0x'
          // set value with invalid opcode
          bytecode += opcode.code.toString('hex')
          if (opcode.programBytesConsumed > 0) {
            bytecode += '00'.repeat(opcode.programBytesConsumed) //PUSHX X_zero_bytes
          }
          // set address
          bytecode += Opcode.PUSH20.code.toString('hex')
          bytecode += remove0x(executionManagerAddress) //PUSH20 Execution Manager address
          // set gas
          bytecode += Opcode.PUSH32.code.toString('hex')
          bytecode += '11'.repeat(32) //PUSH32 0x11...11
          // CALL
          bytecode += Opcode.CALL.code.toString('hex')
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            false,
            `Bytecode containing invalid CALL using ${opcode.name} to set value should have failed!`
          )
        }
      }).timeout(20_000)
      it(`rejects invalid CALLs using opcodes other than PUSH20 to set address`, async () => {
        const invalidAddressSetters: EVMOpcode[] = whitelistedNotHaltingOrCALL.filter(
          (x) => x.name !== 'PUSH20'
        )
        log.debug(
          `Invalid Address Setters ${invalidAddressSetters.map((x) => x.name)}`
        )
        // test for whitelisted, non-halting opcodes (excluding PUSH20)
        for (const opcode of invalidAddressSetters) {
          let bytecode: string = '0x'
          // set value
          bytecode += Opcode.PUSH1.code.toString('hex')
          bytecode += '00' //PUSH1 0x00
          // set address with invalid opcode
          bytecode += opcode.code.toString('hex')
          if (opcode.programBytesConsumed > 0) {
            bytecode += '00'.repeat(opcode.programBytesConsumed) //PUSHX X_zero_bytes
          }
          // set gas
          bytecode += Opcode.PUSH32.code.toString('hex')
          bytecode += '11'.repeat(32) //PUSH32 0x11...11
          // CALL
          bytecode += Opcode.CALL.code.toString('hex')
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            false,
            `Bytecode containing invalid CALL using ${opcode.name} to set value should have failed!`
          )
        }
      }).timeout(20_000)
      it(`rejects invalid CALL with a non-zero value`, async () => {
        let bytecode: string = '0x'
        // set a non-zero value
        bytecode += Opcode.PUSH1.code.toString('hex')
        bytecode += '01' //PUSH1 0x01
        // set address
        bytecode += Opcode.PUSH20.code.toString('hex')
        bytecode += remove0x(executionManagerAddress) //PUSH20 Execution Manager address
        // set gas
        bytecode += Opcode.PUSH32.code.toString('hex')
        bytecode += '11'.repeat(32) //PUSH32 0x11...11
        // CALL
        bytecode += Opcode.CALL.code.toString('hex')
        const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
        res.should.eq(
          false,
          `Bytecode containing invalid CALL PUSH1ing non-zero value should have failed!`
        )
      })
      it(`rejects invalid CALL to a non-Execution Manager address`, async () => {
        let bytecode: string = '0x'
        // set value
        bytecode += Opcode.PUSH1.code.toString('hex')
        bytecode += '00' //PUSH1 0x00
        // set a non-Execution Manager address
        bytecode += Opcode.PUSH20.code.toString('hex')
        bytecode += 'ff'.repeat(20) //PUSH20 invalid address
        // set gas
        bytecode += Opcode.PUSH32.code.toString('hex')
        bytecode += '11'.repeat(32) //PUSH32 0x11...11
        // CALL
        bytecode += Opcode.CALL.code.toString('hex')
        const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
        res.should.eq(
          false,
          `Bytecode containing invalid CALL PUSH20ing non-Execution Manager address should have failed!`
        )
      })
      it(`rejects invalid CALL with only 2 preceding opcodes`, async () => {
        let bytecode: string = '0x'
        // set address
        bytecode += Opcode.PUSH20.code.toString('hex')
        bytecode += remove0x(executionManagerAddress) //PUSH20 Execution Manager address
        // set gas
        bytecode += Opcode.PUSH32.code.toString('hex')
        bytecode += '11'.repeat(32) //PUSH32 0x11...11
        // CALL
        bytecode += Opcode.CALL.code.toString('hex')
        const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
        res.should.eq(
          false,
          `Bytecode containing invalid CALL with only two preceding opcodes should have failed!`
        )
      })
    })
  })
})
