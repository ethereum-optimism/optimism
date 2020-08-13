import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, add0x, remove0x } from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  DEFAULT_UNSAFE_OPCODES,
  EVMOpcode,
  Opcode,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  SYNTHETIX_BYTECODE,
} from '../../test-helpers'

/* Logging */
const log = getLogger('safety-checker', true)

/* Helpers */
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
    x.name !== 'CALLER'
)

/* Tests */
describe.only('Safety Checker', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let SafetyChecker: ContractFactory
  before(async () => {
    SafetyChecker = await ethers.getContractFactory('SafetyChecker')
  })

  let safetyChecker: Contract
  beforeEach(async () => {
    safetyChecker = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'SafetyChecker',
      {
        factory: SafetyChecker,
        params: [
          resolver.addressResolver.address,
        ],
      }
    )

    await resolver.addressResolver.setAddress(
      'ExecutionManager',
      executionManagerAddress
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
          let bytecode: string = '0x'
          bytecode += Opcode.JUMP.code.toString('hex')
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
      it(`accepts valid call`, async () => {
        let bytecode: string = '0x'
        // set address
        bytecode += Opcode.CALLER.code.toString('hex')
        // set value
        bytecode += Opcode.PUSH1.code.toString('hex')
        bytecode += '00' //PUSH1 0x00
        // swap1
        bytecode += Opcode.SWAP1.code.toString('hex')
        // set gas
        bytecode += Opcode.GAS.code.toString('hex')
        // CALL
        bytecode += Opcode.CALL.code.toString('hex')
        const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
        res.should.eq(true, `Bytecode containing valid CALL failed!`)
      })
      it(`rejects invalid CALLs using opcodes other than GAS to set gas`, async () => {
        const invalidGasSetters: EVMOpcode[] = whitelistedNotHaltingOrCALL.filter(
          (x) => x.name !== 'GAS'
        )
        log.debug(`Invalid Gas Setters ${invalidGasSetters.map((x) => x.name)}`)
        // test for whitelisted, non-halting opcodes (excluding PUSHes or DUPs)
        for (const opcode of invalidGasSetters) {
          let bytecode: string = '0x'
          // set value
          bytecode += Opcode.PUSH1.code.toString('hex')
          bytecode += '00' //PUSH1 0x00
          // set address
          bytecode += Opcode.CALLER.code.toString('hex')
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
          bytecode += Opcode.CALLER.code.toString('hex')
          // set gas
          bytecode += Opcode.GAS.code.toString('hex')
          // CALL
          bytecode += Opcode.CALL.code.toString('hex')
          const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
          res.should.eq(
            false,
            `Bytecode containing invalid CALL using ${opcode.name} to set value should have failed!`
          )
        }
      }).timeout(20_000)
      it(`rejects invalid CALLs using opcodes other than CALLER to set address`, async () => {
        const invalidAddressSetters: EVMOpcode[] = whitelistedNotHaltingOrCALL.filter(
          (x) => x.name !== 'CALLER'
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
          bytecode += Opcode.GAS.code.toString('hex')
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
        bytecode += Opcode.CALLER.code.toString('hex')
        // set gas
        bytecode += Opcode.GAS.code.toString('hex')
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
        bytecode += Opcode.GAS.code.toString('hex')
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
        // set addr
        bytecode += Opcode.CALLER.code.toString('hex')
        // set gas
        bytecode += Opcode.GAS.code.toString('hex')
        // CALL
        bytecode += Opcode.CALL.code.toString('hex')
        const res: boolean = await safetyChecker.isBytecodeSafe(bytecode)
        res.should.eq(
          false,
          `Bytecode containing invalid CALL with only two preceding opcodes should have failed!`
        )
      })
    })
    describe('Synthetix contracts', async () => {
      for (const [name, json] of Object.entries(SYNTHETIX_BYTECODE)) {
        //if (name === 'Synthetix.json') {
          it(`${name}: gas cost for init code safety check`, async () => {
            const data = await safetyChecker.interface.encodeFunctionData(
              'isBytecodeSafe',
              [json.bytecode]
            )
            const tx = {
              to: safetyChecker.address,
              data,
            }

            console.log(
              `${name}: initcode is ${json.bytecode.length / 2} bytes long`
            )

            // THIS IS THE NUMBER WE WANT TO GO DOWN--average per-byte cost of a safety check should go down.
            const res = await safetyChecker.provider.estimateGas(tx)
            console.log(`${name}: estimate gas result for initcode: ${res}`)

            const isSafe: boolean = await safetyChecker.isBytecodeSafe(
              json.bytecode
            )
            isSafe.should.eq(true, `Initcode for ${name} should be safe!`)
          })
          it(`${name}: gas cost for deployed bytecode safety check`, async () => {
            const data = await safetyChecker.interface.encodeFunctionData(
              'isBytecodeSafe',
              [json.deployedBytecode]
            )
            const tx = {
              to: safetyChecker.address,
              data,
            }
            console.log(
              `${name}: deployed bytecode is ${json.bytecode.length /
                2} bytes long`
            )

            const res = await safetyChecker.provider.estimateGas(tx)
            console.log(
              `${name}: estimate gas result for deployed bytecode: ${res}`
            )

            const isSafe: boolean = await safetyChecker.isBytecodeSafe(
              json.deployedBytecode
            )
            isSafe.should.eq(true, `Deployed bytecode for ${name} should be safe!`)
          })
        //}
      }
    })
  })
})
