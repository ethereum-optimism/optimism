const fs = require('fs')
const gulp = require('gulp')
const mkdirp = require('mkdirp')
const vyperjs = require('@pigi/vyper-js')

const source = 'src/contracts'
const dest = 'src/compiled'
const contracts = fs.readdirSync(source)

const createExportFile = (name, compiled) => {
  return `export const compiled${name} = ${JSON.stringify(compiled)}`
}

const createIndexFile = () => {
  return contracts.reduce((index, contract) => {
    const name = contract.replace('.vy', '')
    return index + `export * from './compiled${name}'\n`
  }, '')
}

contracts.forEach((contract) => {
  gulp.task(contract, async () => {
    mkdirp.sync(dest)
    const name = contract.replace('.vy', '')
    const compiled = await vyperjs.compile(`${source}/${contract}`)
    await fs.writeFileSync(`${dest}/compiled${name}.ts`, createExportFile(name, compiled))
    await fs.writeFileSync(`${dest}/index.ts`, createIndexFile())
  })
})

gulp.task('compile', gulp.series(contracts))
