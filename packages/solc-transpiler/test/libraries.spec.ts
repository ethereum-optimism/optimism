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
  getLogger,
} from '@eth-optimism/core-utils'
import {
  createMockProvider,
  getWallets,
  deployContract,
} from '@eth-optimism/rollup-full-node'
import { link } from 'ethereum-waffle'

import * as path from 'path'
import * as fs from 'fs'

/* Internal Imports */
import { compile } from '../src'
import { formatBytecode, bufferToBytecode } from '../../rollup-core/build'

const log = getLogger('library-use-compilation')

const safeMathUserPath = path.resolve(
  __dirname,
  './contracts/library/SafeMathUser.sol'
)
const simpleSafeMathPath = path.resolve(
  __dirname,
  './contracts/library/SimpleSafeMath.sol'
)

describe.only('Library usage tests', () => {
  let config
  before(() => {
    config = {
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
        executionManagerAddress: '0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA',
      },
    }
  })

  let provider
  let wallet

  beforeEach(async () => {
    // NOTE: if we run this test in isolation on default port, it works, but in multi-package tests it fails.
    // Hypothesis for why this is: multi-package tests are run in parallel, so we need to use a separate port per package.
    provider = await createMockProvider(9998)
    const wallets = getWallets(provider)
    wallet = wallets[0]
  })
  afterEach(async () => {
    await provider.closeOVM()
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
    const libraryJSON = wrappedSolcJson['contracts']['SimpleSafeMath.sol']['SimpleSafeMath']
    const libUserJSON = wrappedSolcJson['contracts']['SafeMathUser.sol']['SafeMathUser']

    log.debug(`compiler output lib user JSON: \n${JSON.stringify(wrappedSolcJson['contracts']['SafeMathUser.sol'])}`)

    const deployedLibrary = await deployContract(
      wallet,
      libraryJSON,
      [],
      []
    )
    log.debug(`deployed library to: ${deployedLibrary.address}`)
    log.debug(`pre link libuser: ${libUserJSON.evm.bytecode.object}`)
    link(libUserJSON, 'SimpleSafeMath.sol:SimpleSafeMath', deployedLibrary.address)
    const deployedLibUser = await deployContract(wallet, libUserJSON, [], [])
    log.debug(`deployed library user to: ${deployedLibUser.address}`)
    log.debug(`raw linked libuser bytecode is: ${libUserJSON.evm.bytecode.object}`)
    log.debug(`formatted linked libuser bytecode is: ${formatBytecode(bufferToBytecode(hexStrToBuf(libUserJSON.evm.bytecode.object)))}`)

    const returnedUsingLib = await deployedLibUser.use()
    console.log(JSON.stringify(returnedUsingLib))
    returnedUsingLib._hex.should.equal('0x05')
  }).timeout(10000)
})
