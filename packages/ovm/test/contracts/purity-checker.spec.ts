// import '../setup'

// /* External Imports */
// import { EVMOpcode, Opcode } from '@pigi/rollup-core'
// import { getLogger } from '@pigi/core-utils'

// import { Contract } from 'ethers'
// import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

// /* Logging */
// const log = getLogger('test:debug:rollup-list')

// /* Contract Imports */
// import * as PurityChecker from '../../build/contracts/PurityChecker.json'

// const whitelistMask =
//   '0x600800000000000000000000ffffffffffffffff0bcf0000620000013fff0fff'
// const notWhitelisted: EVMOpcode[] = [
//   Opcode.ADDRESS,
//   Opcode.BALANCE,
//   Opcode.BLOCKHASH,
//   Opcode.CALL,
//   Opcode.CALLCODE,
//   Opcode.CALLDATACOPY,
//   Opcode.CALLDATALOAD,
//   Opcode.CALLDATASIZE,
//   Opcode.CALLER,
//   Opcode.CALLVALUE,
//   Opcode.CODESIZE,
//   Opcode.COINBASE,
//   Opcode.CREATE,
//   Opcode.CREATE2,
//   Opcode.DELEGATECALL,
//   Opcode.DIFFICULTY,
//   Opcode.EXTCODECOPY,
//   Opcode.EXTCODESIZE,
//   Opcode.GAS,
//   Opcode.GASLIMIT,
//   Opcode.GASPRICE,
//   Opcode.LOG0,
//   Opcode.LOG1,
//   Opcode.LOG2,
//   Opcode.LOG3,
//   Opcode.LOG4,
//   Opcode.NUMBER,
//   Opcode.ORIGIN,
//   Opcode.SELFDESTRUCT,
//   Opcode.SLOAD,
//   Opcode.SSTORE,
//   Opcode.STATICCALL,
//   Opcode.TIMESTAMP,
// ]
// const whitelisted: EVMOpcode[] = Opcode.ALL_OP_CODES.filter(
//   (x) => notWhitelisted.indexOf(x) < 0
// )

// describe('Purity Checker', () => {
//   const provider = createMockProvider()
//   const [wallet] = getWallets(provider)
//   let purityChecker: Contract

//   /* Deploy a new whitelist contract before each test */
//   beforeEach(async () => {
//     purityChecker = await deployContract(
//       wallet,
//       PurityChecker,
//       [whitelistMask],
//       { gasLimit: 6700000 }
//     )
//   })

//   describe('isBytecodeWhitelisted()', async () => {
//     describe('Empty case', () => {
//       it('should work for empty case', async () => {
//         const res: boolean = await purityChecker.isBytecodePure([])
//         res.should.eq(true, `empty bytecode should be whitelisted!`)
//       })
//     })

//     describe('Single op-code cases', async () => {
//       it('should correctly classify non-whitelisted', async () => {
//         for (const opcode of notWhitelisted) {
//           const res: boolean = await purityChecker.isBytecodePure(
//             `0x${opcode.code.toString('hex')}`
//           )
//           res.should.eq(
//             false,
//             `Opcode ${opcode.name} whitelisted by contract when it shouldn't be!`
//           )
//         }
//       })

//       it('should correctly classify whitelisted', async () => {
//         for (const opcode of whitelisted) {
//           const res: boolean = await purityChecker.isBytecodePure(
//             `0x${opcode.code.toString('hex')}`
//           )
//           res.should.eq(
//             true,
//             `Opcode ${opcode.name} not whitelisted by contract when it should be!`
//           )
//         }
//       })
//     })

//     describe('PUSH cases', async () => {
//       it('skips at least specified number of bytes for PUSH cases', async () => {
//         const invalidOpcode: string = notWhitelisted[0].code.toString('hex')
//         const push1Code: number = parseInt(
//           Opcode.PUSH1.code.toString('hex'),
//           16
//         )
//         for (let i = 1; i < 33; i++) {
//           const bytecode: string = `0x${Buffer.of(push1Code + i - 1).toString(
//             'hex'
//           )}${invalidOpcode.repeat(i)}`
//           const res: boolean = await purityChecker.isBytecodePure(bytecode)
//           res.should.eq(
//             true,
//             `PUSH${i} failed by not skipping ${i} bytes of bytecode!`
//           )
//         }
//       })

//       it('skips at most specified number of bytes for PUSH cases', async () => {
//         const invalidOpcode: string = notWhitelisted[0].code.toString('hex')
//         const push1Code: number = parseInt(
//           Opcode.PUSH1.code.toString('hex'),
//           16
//         )
//         for (let i = 1; i < 33; i++) {
//           const bytecode: string = `0x${Buffer.of(push1Code + i - 1).toString(
//             'hex'
//           )}${invalidOpcode.repeat(i + 1)}`
//           const res: boolean = await purityChecker.isBytecodePure(bytecode)
//           res.should.eq(
//             false,
//             `PUSH${i} succeeded, skipping ${i +
//               1} bytes of bytecode when it should have skipped ${i} bytes!`
//           )
//         }
//       })
//     })

//     describe('multiple opcode cases', async () => {
//       it('works for whitelisted codes', async () => {
//         let bytecode: string = '0x'
//         const invalidOpcode: string = notWhitelisted[0].code.toString('hex')

//         for (const opcode of whitelisted) {
//           bytecode += `${opcode.code.toString('hex')}${invalidOpcode.repeat(
//             opcode.programBytesConsumed
//           )}`
//         }

//         const res: boolean = await purityChecker.isBytecodePure(bytecode)
//         res.should.eq(true, `Bytecode of all whitelisted failed!`)
//       })

//       it('fails for whitelisted codes with one not on whitelist at the end', async () => {
//         let bytecode: string = '0x'
//         const invalidOpcode: string = notWhitelisted[0].code.toString('hex')

//         for (const opcode of whitelisted) {
//           bytecode += `${opcode.code.toString('hex')}${invalidOpcode.repeat(
//             opcode.programBytesConsumed
//           )}`
//         }

//         for (const opcode of notWhitelisted) {
//           const res: boolean = await purityChecker.isBytecodePure(
//             bytecode + opcode.code.toString('hex')
//           )
//           res.should.eq(
//             false,
//             `Bytecode of all whitelisted + ${opcode.name} passed when it should have failed!`
//           )
//         }
//       }).timeout(20_000)
//     })
//   })
// })
