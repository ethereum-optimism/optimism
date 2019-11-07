/* External Imports */
import fs = require('fs')
import path = require('path')
import chai = require('chai')
import chaiAsPromised = require('chai-as-promised')

/* Internal Imports */
import { rootPath } from '../index'

chai.use(chaiAsPromised)
const should = chai.should()

const testArtifactsDir = path.join(rootPath, 'test', 'artifacts.test.tmp')
const testRootPath = path.join(testArtifactsDir, (+new Date()).toString())

// If these directories don't exist, create them.
fs.mkdirSync(testArtifactsDir, { recursive: true })
fs.mkdirSync(testRootPath, { recursive: true })

export { should }
export { testRootPath as rootPath }
