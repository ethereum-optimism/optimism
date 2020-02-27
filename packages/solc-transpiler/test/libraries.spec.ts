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
  remove0x,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import { createMockProvider, getWallets, deployContract,  } from '@eth-optimism/rollup-full-node'
import { link } from 'ethereum-waffle'


import * as path from 'path'
import * as fs from 'fs'

/* Internal Imports */
import { compile } from '../src'

const safeMathUserPath = path.resolve(__dirname, './contracts/library/SafeMathUser.sol')
const simpleSafeMathPath = path.resolve(__dirname, './contracts/library/SimpleSafeMath.sol')
const config = {
  language: 'Solidity',
  sources: {
    'SafeMathUser.sol': {
      content: fs.readFileSync(safeMathUserPath, 'utf8'),
    },
    'SimpleSafeMath.sol': {
      content: fs.readFileSync(simpleSafeMathPath, 'utf8'),
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

describe('Library usage tests', () => {
//   const transpiler: Transpiler = new TranspilerImpl(
//     new OpcodeWhitelistImpl(),
//     new OpcodeReplacerImpl(ZERO_ADDRESS)
//   )
//   console.log(`replacing via: ${JSON.stringify(new OpcodeReplacerImpl(ZERO_ADDRESS))}`)

  let provider
  let wallet

  beforeEach(async () => {
    provider = await createMockProvider()
    const wallets = getWallets(provider)
    wallet = wallets[0]
  })
  afterEach(async () => {
      provider.closeOVM()
  })

//   const getContractTranspiledBytecode = (
//     compiledJson: any,
//     isDeployedBytecode: boolean = false
//   ): string => {
//     const auxData = compiledJson.evm.legacyAssembly['.data']['0']['.auxdata']
//     const bytecode = isDeployedBytecode
//       ? compiledJson.evm.deployedBytecode.object
//       : compiledJson.evm.bytecode.object

//     // Remove the AuxData at the end of the contract bytecode because this may be different even if it's the exact same contract
//     const bytecodeWithoutAuxdata: string = bytecode.split(auxData)[0]
//     const transpilationResult: TranspilationResult = transpiler.transpileRawBytecode(
//       hexStrToBuf(bytecodeWithoutAuxdata)
//     )

//     transpilationResult.succeeded.should.eq(
//       true,
//       'Cannot evaluate test because transpiling solc bytecode failed.'
//     )

//     return remove0x(
//       bufToHexString((transpilationResult as SuccessfulTranspilation).bytecode)
//     )
//   }

  it('should allow us to transpile, link, and use libraries', async () => {
    const wrappedSolcResult = compile(JSON.stringify(config))
    const wrappedSolcJson = JSON.parse(wrappedSolcResult)
    console.log(`wrapped solc json is: ${JSON.stringify(wrappedSolcJson)}`)

    const deployedLibrary = await deployContract(wallet, wrappedSolcJson['contracts']['SimpleSafeMath.sol']['SimpleSafeMath'], [], [])
    // console.log(`deployed library to: ${deployedLibrary.address}`)
    // const libUser = wrappedSolcJson['contracts']['SafeMathUser.sol']['SafeMathUser']
    // link(libUser, 'SimpleSafeMath.sol:SimpleSafeMath', deployedLibrary.address)
    // const deployedLibUser = await deployContract(wallet, libUser, [], [])
    // console.log(`deployed library user to: ${deployedLibUser.address}`)

    // const returnedUsingLib = await deployedLibUser.use()
    // returnedUsingLib.should.equal(2 + 3)


    // const waffleTranspiledDeployedBytecode: string = getContractTranspiledBytecode(
    //   SafeMathUser,
    //   true
    // )
    // const wrappedSolcTranspiledDeployedBytecode: string =
    //   wrappedSolcJson['contracts']['SafeMathuUser.sol']['SafeMathUser'].evm.deployedBytecode
    //     .object

    // waffleTranspiledDeployedBytecode.should.eq(
    //   wrappedSolcTranspiledDeployedBytecode,
    //   'Transpiled deployed bytecode mismatch!'
    // )
  }).timeout(10000)
})
