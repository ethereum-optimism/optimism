/* External Imports */
import path = require('path')

const rootPath = __dirname
const dbRootPath = path.join(__dirname, 'db')

export { rootPath, dbRootPath }
export * from './src/app'
export * from './src/interfaces'
