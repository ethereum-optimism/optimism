import { expect } from '@eth-optimism/core-utils/test/setup'
import fs from 'fs/promises'
import path from 'path'
import { isBytecodeERC20 } from '../scripts/utils'

describe('Utils', () => {
  // Read in the mock data
  const contracts = {}
  before(async () => {
    const files = await fs.readdir(path.join(__dirname, 'data'))
    for (const filename of files) {
      const file = await fs.readFile(path.join(__dirname, 'data', filename))
      const name = path.parse(filename).name
      const json = JSON.parse(file.toString())
      contracts[name] = {
        bytecode: json.bytecode.toString().trim(),
        expected: json.expected,
      }
    }
  })

  it('isBytecodeERC20', () => {
    for (const [name, contract] of Object.entries(contracts)) {
      describe(`contract ${name}`, () => {
        it('should be identified erc20', () => {
          const result = isBytecodeERC20((contract as any).bytecode as string)
          expect(result).to.eq((contract as any).expected)
        })
      })
    }
  })
})
