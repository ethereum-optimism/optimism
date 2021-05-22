const os = require('os')

// this script unbundles the published packages output
// from changesets action to a key-value pair to be used
// with our publishing CI workflow
data = process.argv[2]
data = JSON.parse(data)

builder = false
for (const i of data) {
  const name = i.name.replace("@eth-optimism/", "")
  if (name !== 'l2geth') {
    builder = true
  }
  const version = i.version
  process.stdout.write(`::set-output name=${name}::${version}` + os.EOL)
}

if (builder) {
  process.stdout.write(`::set-output name=builder::true` + os.EOL)
}

