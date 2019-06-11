/* External Imports */
import path = require('path')

const rootPath = __dirname
const dbRootPath = path.join(__dirname, 'db')

export * from './src/interfaces'
export { rootPath, dbRootPath }
