const os = require('os')

// this script unbundles the published packages output
// from changesets action to a key-value pair to be used
// with our publishing CI workflow
data = process.argv[2]
data = JSON.parse(data)

process.stdout.write(`::set-output name=fault-detector::0.0.0-20230123180301` + os.EOL)
for (const i of data) {
  const name = i.name.replace("@eth-optimism/", "")
  const version = i.version
  process.stdout.write(`::set-output name=${name}::${version}` + os.EOL)
}
