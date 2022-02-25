import fs from 'fs'
import path from 'path'

import chai = require('chai')
import Mocha from 'mocha'
import chaiAsPromised from 'chai-as-promised'
import { BigNumber } from 'ethers'

// Chai plugins go here.
chai.use(chaiAsPromised)

const should = chai.should()
const expect = chai.expect

const readMockData = () => {
  const mockDataPath = path.join(__dirname, 'unit-tests', 'examples')
  const paths = fs.readdirSync(mockDataPath)
  const files = []
  for (const filename of paths) {
    // Skip non .txt files
    if (!filename.endsWith('.txt')) {
      continue
    }
    const filePath = path.join(mockDataPath, filename)
    const file = fs.readFileSync(filePath)
    const obj = JSON.parse(file.toString())
    // Reserialize the BigNumbers
    obj.input.extraData.prevTotalElements = BigNumber.from(
      obj.input.extraData.prevTotalElements
    )
    obj.input.extraData.batchIndex = BigNumber.from(
      obj.input.extraData.batchIndex
    )
    if (obj.input.event.args.length !== 3) {
      throw new Error(`ABI mismatch`)
    }
    obj.input.event.args = obj.input.event.args.map(BigNumber.from)
    obj.input.event.args._startingQueueIndex = obj.input.event.args[0]
    obj.input.event.args._numQueueElements = obj.input.event.args[1]
    obj.input.event.args._totalElements = obj.input.event.args[2]
    obj.input.extraData.batchSize = BigNumber.from(
      obj.input.extraData.batchSize
    )
    files.push(obj)
  }
  return files
}

export { should, expect, Mocha, readMockData }
