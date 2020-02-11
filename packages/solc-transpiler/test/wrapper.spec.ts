import './setup'

/* External Imports */
import {
  OpcodeReplacerImpl,
  OpcodeWhitelistImpl,
  SuccessfulTranspilation,
  TranspilationResult,
  Transpiler,
  TranspilerImpl,
} from '@eth-optimism/rollup-dev-tools'
import {
  bufToHexString,
  hexStrToBuf,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import * as path from 'path'
import * as fs from 'fs'

/* Internal Imports */
import { compile } from '../src'
import * as DummyContract from './contracts/build/Dummy.json'
import * as Dummy2Contract from './contracts/build/Dummy2.json'
import * as Dummy3Contract from './contracts/build/Dummy3.json'

const dummyPath = path.resolve(__dirname, './contracts/Dummy.sol')
const dummy2Path = path.resolve(__dirname, './contracts/Dummy2.sol')
const config = {
  language: 'Solidity',
  sources: {
    'Dummy.sol': {
      content: fs.readFileSync(dummyPath, 'utf8'),
    },
  },
  settings: {
    outputSelection: {
      '*': {
        '*': ['*'],
      },
    },
    executionManagerAddress: ZERO_ADDRESS,
  },
}

const multiConfig = { ...config }
multiConfig.sources['Dummy2.sol'] = {
  content: fs.readFileSync(dummy2Path, 'utf8'),
}

const configWithoutLegacyAssembly = { ...config }
configWithoutLegacyAssembly.settings.outputSelection['*']['*'] = [
  'abi',
  'evm.bytecode',
  'evm.deployedBytecode',
]

describe('Wrapper tests', () => {
  const transpiler: Transpiler = new TranspilerImpl(
    new OpcodeWhitelistImpl(),
    new OpcodeReplacerImpl(ZERO_ADDRESS)
  )

  const getContractTranspiledBytecode = (
    compiledJson: any,
    isDeployedBytecode: boolean = false
  ): string => {
    const auxData = compiledJson.evm.legacyAssembly['.data']['0']['.auxdata']
    const bytecode = isDeployedBytecode
      ? compiledJson.evm.deployedBytecode.object
      : compiledJson.evm.bytecode.object

    // Remove the AuxData at the end of the contract bytecode because this may be different even if it's the exact same contract
    const bytecodeWithoutAuxdata: string = bytecode.split(auxData)[0]
    const transpilationResult: TranspilationResult = transpiler.transpileRawBytecode(
      hexStrToBuf(bytecodeWithoutAuxdata)
    )

    transpilationResult.succeeded.should.eq(
      true,
      'Cannot evaluate test because transpiling solc bytecode failed.'
    )

    return bufToHexString(
      (transpilationResult as SuccessfulTranspilation).bytecode
    )
  }

  it('should match deployed bytecode transpilation result if compiled and transpiled separately', () => {
    const wrappedSolcResult = compile(JSON.stringify(config))
    const wrappedSolcJson = JSON.parse(wrappedSolcResult)

    const waffleTranspiledDeployedBytecode: string = getContractTranspiledBytecode(
      DummyContract,
      true
    )
    const wrappedSolcTranspiledDeployedBytecode: string =
      wrappedSolcJson['contracts']['Dummy.sol']['Dummy'].evm.deployedBytecode
        .object

    waffleTranspiledDeployedBytecode.should.eq(
      wrappedSolcTranspiledDeployedBytecode,
      'Transpiled deployed bytecode mismatch!'
    )
  }).timeout(10000)

  it('should work for multiple sources', () => {
    const wrappedSolcResult = compile(JSON.stringify(multiConfig))
    const wrappedSolcJson = JSON.parse(wrappedSolcResult)

    const dummyWaffleTranspiledDeployedBytecode: string = getContractTranspiledBytecode(
      DummyContract,
      true
    )
    const dummyWrappedSolcTranspiledDeployedBytecode: string =
      wrappedSolcJson['contracts']['Dummy.sol']['Dummy'].evm.deployedBytecode
        .object

    dummyWaffleTranspiledDeployedBytecode.should.eq(
      dummyWrappedSolcTranspiledDeployedBytecode,
      'Dummy transpiled deployed bytecode mismatch!'
    )

    const dummy2WaffleTranspiledDeployedBytecode: string = getContractTranspiledBytecode(
      Dummy2Contract,
      true
    )
    const dummy2WrappedSolcTranspiledDeployedBytecode: string =
      wrappedSolcJson['contracts']['Dummy2.sol']['Dummy2'].evm.deployedBytecode
        .object

    dummy2WaffleTranspiledDeployedBytecode.should.eq(
      dummy2WrappedSolcTranspiledDeployedBytecode,
      'Dummy2 transpiled deployed bytecode mismatch!'
    )

    const dummy3WaffleTranspiledDeployedBytecode: string = getContractTranspiledBytecode(
      Dummy3Contract,
      true
    )
    const dummy3WrappedSolcTranspiledDeployedBytecode: string =
      wrappedSolcJson['contracts']['Dummy2.sol']['Dummy3'].evm.deployedBytecode
        .object

    dummy3WaffleTranspiledDeployedBytecode.should.eq(
      dummy3WrappedSolcTranspiledDeployedBytecode,
      'Dummy3 transpiled deployed bytecode mismatch!'
    )
  })

  it('should work without `evm.legacyAssembly` outputSelection', () => {
    const wrappedSolcResult = compile(
      JSON.stringify(configWithoutLegacyAssembly)
    )
    const wrappedSolcJson = JSON.parse(wrappedSolcResult)

    const waffleTranspiledDeployedBytecode: string = getContractTranspiledBytecode(
      DummyContract,
      true
    )
    const wrappedSolcTranspiledDeployedBytecode: string =
      wrappedSolcJson['contracts']['Dummy.sol']['Dummy'].evm.deployedBytecode
        .object

    waffleTranspiledDeployedBytecode.should.eq(
      wrappedSolcTranspiledDeployedBytecode,
      'Transpiled deployed bytecode mismatch!'
    )

    const hasLegacyAssembly: boolean =
      'legacyAssembly' in wrappedSolcJson['contracts']['Dummy.sol']['Dummy'].evm
    hasLegacyAssembly.should.equal(
      false,
      'Legacy assembly should not be present in output!'
    )
  })
})
