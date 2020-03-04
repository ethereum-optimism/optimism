import './setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
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

const log = getLogger('library-use-compilation')

const safeMathUserPath = path.resolve(
  __dirname,
  './contracts/library/SafeMathUser.sol'
)
const simpleSafeMathPath = path.resolve(
  __dirname,
  './contracts/library/SimpleSafeMath.sol'
)
const simpleUnsafeMathPath = path.resolve(
  __dirname,
  './contracts/library/SimpleUnsafeMath.sol'
)

describe('Library usage tests', () => {
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
        'SimpleUnsafeMath.sol': {
          content: fs.readFileSync(simpleUnsafeMathPath, 'utf8'),
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
  let deployedLibUser
  beforeEach(async function() {
    this.timeout(20000)
    // NOTE: if we run this test in isolation on default port, it works, but in multi-package tests it fails.
    // Hypothesis for why this is: multi-package tests are run in parallel, so we need to use a separate port per package.
    provider = await createMockProvider(9998)
    const wallets = getWallets(provider)
    wallet = wallets[0]

    const wrappedSolcResult = compile(JSON.stringify(config))
    const wrappedSolcJson = JSON.parse(wrappedSolcResult)
    const simpleSafeMathJSON =
      wrappedSolcJson['contracts']['SimpleSafeMath.sol']['SimpleSafeMath']
    const simpleUnsafeMathJSON =
      wrappedSolcJson['contracts']['SimpleUnsafeMath.sol']['SimpleUnsafeMath']
    const libUserJSON =
      wrappedSolcJson['contracts']['SafeMathUser.sol']['SafeMathUser']

    // Deploy and link safe math
    const deployedSafeMath = await deployContract(
      wallet,
      simpleSafeMathJSON,
      [],
      []
    )
    log.debug(`deployed SimpleSafeMath to: ${deployedSafeMath.address}`)
    link(
      libUserJSON,
      'SimpleSafeMath.sol:SimpleSafeMath',
      deployedSafeMath.address
    )

    // Deoloy and link unsafe math
    const deployedUnsafeMath = await deployContract(
      wallet,
      simpleUnsafeMathJSON,
      [],
      []
    )
    log.debug(`deployed UnsafeMath to: ${deployedUnsafeMath.address}`)
    log.debug(`before second link: ${JSON.stringify(libUserJSON)}`)
    link(
      libUserJSON,
      'SimpleUnsafeMath.sol:SimpleUnsafeMath',
      deployedUnsafeMath.address
    )

    // Deploy library user
    deployedLibUser = await deployContract(wallet, libUserJSON, [], [])
    log.debug(`deployed library user to: ${deployedLibUser.address}`)
  })
  afterEach(async () => {
    await provider.closeOVM()
  })

  it('should allow us to transpile, link, and query contract methods which use a single library', async () => {
    const returnedUsingLib = await deployedLibUser.useLib()
    returnedUsingLib._hex.should.equal('0x05')
  })

  it('should allow us to transpile, link, and query contract methods which use a multiple libraries', async () => {
    const returnedUsingLib = await deployedLibUser.use2Libs()
    returnedUsingLib._hex.should.equal('0x06')
  })
})
