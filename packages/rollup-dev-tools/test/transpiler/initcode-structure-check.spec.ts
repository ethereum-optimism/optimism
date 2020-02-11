/* External Imports */
import { ethers } from 'ethers'
import {
  bufToHexString,
  remove0x,
  getLogger,
  hexStrToBuf,
} from '@eth-optimism/core-utils'
import {
  Address,
  bytecodeToBuffer,
  EVMBytecode,
  EVMOpcode,
  formatBytecode,
  Opcode,
  EVMOpcodeAndBytes,
  bufferToBytecode,
} from '@eth-optimism/rollup-core'

// constants related
import * as ethereumjsAbi from 'ethereumjs-abi'
import * as ConstructorlessWithConstants from '../contracts/build/ConstructorlessWithConstants.json'
import * as StandaloneConstructorWithConstants from '../contracts/build/StandaloneConstructorWithConstants.json'
import * as SubcallConstructorWithoutConstants from '../contracts/build/SubcallConstructorWithoutConstants.json'
import * as SubcallConstructorWithConstants from '../contracts/build/SubcallConstructorWithConstants.json'
import * as SubcallConstructorWithConstantsInBoth from '../contracts/build/SubcallConstructorWithConstantsInBoth.json'
import * as SubcallConstructorWithNestedSubcalls from '../contracts/build/SubcallConstructorWithNestedSubcalls.json'

// constructor params related
import * as ConstructorWithSmallParameter from '../contracts/build/ConstructorWithSmallParameter.json'
import * as ConstructorWithBigParameter from '../contracts/build/ConstructorWithBigParameter.json'
import * as ConstructorWithTwoBigParameters from '../contracts/build/ConstructorWithTwoBigParameters.json'
import * as ConstantsInConstructorAndBody from '../contracts/build/ConstantsInConstructorAndBody.json'

// constructor params accessing constant before accessing those params
import * as ConstructorWithTwoBigParametersAccessingConstantBefore from '../contracts/build/ConstructorWithTwoBigParametersAccessingConstantBefore.json'

/*
    ********
    CONSTRUCTOR STRUCTURE CHECKS
    ********
    These tests were used to investigate how solc creates initcode: specifically, the relation between init logic used in CREATE and the deployed bytecode.
    The results were that initcode consists of the following:
    [deploy logic] [bytecode to deploy] [constants used in the deploy logic]
    Which is now the format assumed for transpiler.transpile().  
    The test asserting the above structure is the only one not .skip()ped, but the remainder have been left for posterity.
    They basically just logged various combinations of Solidity contracts with constructors, subcalls, and constants, so that they could be assesed visually.
*/

const log = getLogger(`constructor-exploration`)
const abi = new ethers.utils.AbiCoder()

interface BytecodeInspection {
  contractName: string
  initcode: Buffer
  deployedBytecode: Buffer
}

const getBytecodeInspection = (json: any, name: string): BytecodeInspection => {
  return {
    contractName: name,
    initcode: hexStrToBuf(json.bytecode),
    deployedBytecode: hexStrToBuf(json.evm.deployedBytecode.object),
  }
}

const constructorContracts: BytecodeInspection[] = [
  getBytecodeInspection(
    ConstructorlessWithConstants,
    'ConstructorlessWithConstants'
  ),
  getBytecodeInspection(
    StandaloneConstructorWithConstants,
    'StandaloneConstructorWithConstants'
  ),
  getBytecodeInspection(
    SubcallConstructorWithoutConstants,
    'SubcallConstructorWithoutConstants'
  ),
  getBytecodeInspection(
    SubcallConstructorWithConstants,
    'SubcallConstructorWithConstants'
  ),
  getBytecodeInspection(
    SubcallConstructorWithConstantsInBoth,
    'SubcallConstructorWithConstantsInBoth'
  ),
  getBytecodeInspection(
    SubcallConstructorWithNestedSubcalls,
    'SubcallConstructorWithNestedSubcalls'
  ),
]

const getInitcodePrefix = (cont: BytecodeInspection): EVMBytecode => {
  const deployedBytecode: Buffer = cont.deployedBytecode
  const deployedBytecodeLength: number = deployedBytecode.byteLength
  const initcodeLength: number = cont.initcode.byteLength
  const initcodePrefix: Buffer = cont.initcode.slice(
    0,
    initcodeLength - deployedBytecodeLength
  )
  return bufferToBytecode(initcodePrefix)
}

describe('Constructor/initcode positioning test', () => {
  it.skip(`let's just look at the hex bytes,`, async () => {
    for (const cont of constructorContracts) {
      log.debug(
        `contract ${cont.contractName} has: \n     initcode: ${bufToHexString(
          cont.initcode
        )}\n     deployed bytecode: ${bufToHexString(cont.deployedBytecode)} \n`
      )
    }
  })
  it(`in all cases the initcode should be a prefixed version of deployedBytecode`, async () => {
    for (const cont of constructorContracts) {
      const deployedBytecode: Buffer = cont.deployedBytecode
      const deployedBytecodeLength: number = deployedBytecode.byteLength
      const initLogicLength: number = cont.initcode.indexOf(deployedBytecode)
      const constantsUsedInConstructorStart: number =
        cont.initcode.indexOf(deployedBytecode) + deployedBytecodeLength
      const sliceOfDeployedBytecodeInInitcode: Buffer = cont.initcode.slice(
        initLogicLength,
        constantsUsedInConstructorStart
      )
      if (
        Buffer.compare(deployedBytecode, sliceOfDeployedBytecodeInInitcode) !==
        0
      ) {
        log.debug(`failed to hold for ${cont.contractName}.`)
        log.debug(`Raw initcode: ${bufToHexString(cont.initcode)}`)
        log.debug(`Raw deployedBytecode: ${bufToHexString(deployedBytecode)}`)
        log.debug(
          `The deployed bytecode is: ${formatBytecode(
            bufferToBytecode(deployedBytecode)
          )}`
        )
        log.debug(
          `The slice of initcode between constructor logic and constants accessed in initcode is: ${formatBytecode(
            bufferToBytecode(sliceOfDeployedBytecodeInInitcode)
          )}`
        )
      }
      deployedBytecode.should.deep.equal(sliceOfDeployedBytecodeInInitcode)
    }
  })

  const constructorParameterContracts: BytecodeInspection[] = [
    getBytecodeInspection(
      ConstructorWithSmallParameter,
      'ConstructorWithSmallParameter'
    ),
    getBytecodeInspection(
      ConstructorWithBigParameter,
      'ConstructorWithBigParameter'
    ),
    getBytecodeInspection(
      ConstructorWithTwoBigParameters,
      'ConstructorWithTwoBigParameters'
    ),
  ]

  it.skip(`let's look at some initcode prefixes which utilize different params`, async () => {
    for (const cont of constructorParameterContracts) {
      const initcodePrefix: EVMBytecode = getInitcodePrefix(cont)
      log.debug(`The initcode prefix for contract ${cont.contractName}.sol is:`)
      log.debug(formatBytecode(initcodePrefix))
    }
  })
  it.skip(`let's make sure that accessing a constant before accessing a constructor input doesn't trigger out JUMPDEST...CODECOPY prematurely`, async () => {
    const crazyConstructorPrefix: EVMBytecode = getInitcodePrefix(
      getBytecodeInspection(
        ConstructorWithTwoBigParametersAccessingConstantBefore,
        'ConstructorWithTwoBigParametersAccessingConstantBefore'
      )
    )
    log.debug(
      `solidity with const accessed in constructor before accessing constructor params gives us the following initcode prefix:`
    )
    log.debug(formatBytecode(crazyConstructorPrefix))
  })
  it.skip(`let's look at a constructor which recieves no input but accesses a constant`, async () => {
    const standaloneConstructorWithConstantsPrefix: EVMBytecode = getInitcodePrefix(
      getBytecodeInspection(
        StandaloneConstructorWithConstants,
        'StandaloneConstructorWithConstants'
      )
    )
    log.debug(
      `solidity with const accessed in constructor and no constructor params gives us the following initcode prefix:`
    )
    log.debug(formatBytecode(standaloneConstructorWithConstantsPrefix))
  })
})
