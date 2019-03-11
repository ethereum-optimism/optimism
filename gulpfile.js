const gulp = require('gulp')
const ts = require('gulp-typescript')
const clean = require('gulp-clean')
const deleteEmpty = require('delete-empty')
const minimist = require('minimist')

const packages = {
  utils: ts.createProject('packages/utils/tsconfig.json'),
}
const modules = Object.keys(packages)
const source = 'packages'
const argv = minimist(process.argv.slice(2))
const dist = argv['dist'] || source
const pkgs = argv['pkgs'] || modules

gulp.task('default', function() {
  modules.forEach((module) => {
    gulp.watch(
      [`${source}/${module}/**/*.ts`, `${source}/${module}/*.ts`],
      [module]
    )
  })
})

gulp.task('copy-misc', function() {
  return gulp
    .src(['README.md', 'LICENSE.txt', '.npmignore'])
    .pipe(gulp.dest(`${source}/utils`))
})

gulp.task('clean:output', function() {
  return gulp
    .src([`${source}/**/*.js`, `${source}/**/*.d.ts`], {
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
      .pipe(gulp.dest(`${dist}/${module}`))
  })
})

gulp.task('build', gulp.series(pkgs))
