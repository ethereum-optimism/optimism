import './setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'

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
    executionManagerAddress: '0x6454c9d69a4721feba60e26a367bd4d56196ee7c',
  },
}

describe('Library usage tests', () => {
  it('should compile with libraries', async () => {
    const wrappedSolcResult = compile(JSON.stringify(config))
    const wrappedSolcJson = JSON.parse(wrappedSolcResult)

    wrappedSolcJson.contracts.should.not.equal(
      undefined,
      'No compiled contracts found!'
    )

    wrappedSolcJson.contracts['SimpleSafeMath.sol'].should.not.equal(
      undefined,
      'SimpleSafeMath file not found!'
    )
    wrappedSolcJson.contracts['SimpleSafeMath.sol'][
      'SimpleSafeMath'
    ].should.not.equal(undefined, 'SimpleSafeMath contract not found!')

    wrappedSolcJson.contracts['SimpleUnsafeMath.sol'].should.not.equal(
      undefined,
      'SimpleUnsafeMath file not found!'
    )
    wrappedSolcJson.contracts['SimpleUnsafeMath.sol'][
      'SimpleUnsafeMath'
    ].should.not.equal(undefined, 'SimpleUnsafeMath contract not found!')

    wrappedSolcJson.contracts['SafeMathUser.sol'].should.not.equal(
      undefined,
      'SafeMathUser file not found!'
    )
    wrappedSolcJson.contracts['SafeMathUser.sol'][
      'SafeMathUser'
    ].should.not.equal(undefined, 'SafeMathUser contract not found!')
  }).timeout(10_000)
})
