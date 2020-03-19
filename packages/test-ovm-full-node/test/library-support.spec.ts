import './setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import {
  deployContract,
  createMockProvider,
  getWallets,
} from '@eth-optimism/rollup-full-node'
import solcTranspiler from '@eth-optimism/solc-transpiler'
import { EXECUTION_MANAGER_ADDRESS } from '@eth-optimism-test/integration-test-utils'
import { link } from 'ethereum-waffle'

import * as path from 'path'
import * as fs from 'fs'

const log = getLogger('library-use-compilation')

const safeMathUserPath = path.resolve(
  __dirname,
  '../contracts/library/SafeMathUser.sol'
)
const simpleSafeMathPath = path.resolve(
  __dirname,
  '../contracts/library/SimpleSafeMath.sol'
)
const simpleUnsafeMathPath = path.resolve(
  __dirname,
  '../contracts/library/SimpleUnsafeMath.sol'
)

const config = {
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
  },
}

process.env.EXECUTION_MANAGER_ADDRESS = EXECUTION_MANAGER_ADDRESS

describe('Library usage tests', () => {
  let provider
  let wallet
  let deployedLibUser
  beforeEach(async function() {
    // NOTE: if we run this test in isolation on default port, it works, but in multi-package tests it fails.
    // Hypothesis for why this is: multi-package tests are run in parallel, so we need to use a separate port per package.
    provider = await createMockProvider(9998)
    const wallets = getWallets(provider)
    wallet = wallets[0]

    const wrappedSolcResult = (solcTranspiler as any).compile(JSON.stringify(config))
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
  }).timeout(20_000)

  it('should allow us to transpile, link, and query contract methods which use a multiple libraries', async () => {
    const returnedUsingLib = await deployedLibUser.use2Libs()
    returnedUsingLib._hex.should.equal('0x06')
  }).timeout(20_000)
})
