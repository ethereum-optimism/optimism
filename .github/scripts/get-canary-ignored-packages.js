const { exec } = require('child_process')

if (process.env['WANTED_PACKAGES'].length == 0) {
  console.log('0')
  return
}

exec('yarn workspaces --json info', (error, stdout, stderr) => {
  if (error) {
    return
  }
  if (stderr) {
    return
  }
  let output = JSON.parse(stdout)
  const wantedPackages = process.env['WANTED_PACKAGES'].split(',')
  let allPackages = Object.keys(JSON.parse(output['data']))
  var ignoredPackages = allPackages.filter((p) => !wantedPackages.includes(p))
  console.log(`${ignoredPackages.toString()}`)
})
