const os = require('os')

// this script unbundles the published packages output
// from changesets action to a key-value pair to be used
// with our publishing CI workflow
data = process.argv[2]
data = JSON.parse(data)

// Packages that do not depend on the builder.
// There are more packages that depend on the
// builder than not, so keep track of this list instead
const nonBuilders = new Set([
  'l2geth',
  'gas-oracle',
  'proxyd',
  'rpc-proxy',
])

builder = false
for (const i of data) {
  const name = i.name.replace("@eth-optimism/", "")
  if (!nonBuilders.has(name)) {
    builder = true
  }
  const version = i.version
  process.stdout.write(`::set-output name=${name}::${version}` + os.EOL)
}

if (builder) {
  process.stdout.write(`::set-output name=use_builder::true` + os.EOL)
}

