const fs = require('fs')
const gulp = require('gulp')
const ts = require('gulp-typescript')
const clean = require('gulp-clean')
const deleteEmpty = require('delete-empty')
const minimist = require('minimist')

const source = 'packages'
const modules = fs.readdirSync(source).filter((item) => {
  return fs.lstatSync(`${source}/${item}`).isDirectory()
})
const packages = modules.reduce((pkgs, module) => {
  pkgs[module] = ts.createProject(`${source}/${module}/tsconfig.json`)
  return pkgs
}, {})

const argv = minimist(process.argv.slice(2))
const dist = argv['dist']
const pkgs = argv['pkgs'] ? argv['pkgs'].split(',') : modules

gulp.task('default', function() {
  modules.forEach((module) => {
    gulp.watch(
      [`${source}/${module}/**/*.ts`, `${source}/${module}/*.ts`],
      [module]
    )
  })
})

gulp.task('copy-misc', function() {
  let pipeline = gulp.src(['LICENSE.txt'])
  modules.forEach((module) => {
    pipeline = pipeline.pipe(gulp.dest(`${source}/${module}`))
  })
  return pipeline
})

gulp.task('clean:output', function() {
  return gulp
    .src([`${source}/**/build`], {
      read: false,
    })
    .pipe(clean())
})

gulp.task('clean:dirs', function(done) {
  deleteEmpty.sync(`${source}/`)
  done()
})

gulp.task('clean:bundle', gulp.series('clean:output', 'clean:dirs'))

modules.forEach((module) => {
  gulp.task(module, () => {
    return packages[module]
      .src()
      .pipe(packages[module]())
      .pipe(gulp.dest(dist ? `${dist}/${module}` : `${source}/${module}/build`))
  })
})

gulp.task('build', gulp.series(pkgs))
