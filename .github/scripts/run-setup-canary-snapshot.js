const { exec } = require('child_process')

var command = 'yarn changeset version --snapshot'
if (process.env['IGNORED_PACKAGES'].length > 0) {
  var commandArguments = ''
  var ignoredPackages = process.env['IGNORED_PACKAGES'].split(',')
  for (let i = 0; i < ignoredPackages.length; i++) {
    commandArguments += ' --ignore ' + ignoredPackages[i]
  }
  command += commandArguments
}

exec(command, (error, stdout, stderr) => {
  if (error) {
    console.log(error)
    return
  }
  if (stderr) {
    console.log(stderr)
    return
  }
  console.log(stdout)
})
