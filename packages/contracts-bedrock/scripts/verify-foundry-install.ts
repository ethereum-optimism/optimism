import { execSync } from 'child_process'

import { sync as commandExistsSync } from 'command-exists'

if (!commandExistsSync('forge')) {
  console.error(
    `Command failed. Is Foundry not installed?
Consider installing via \`curl -L https://foundry.paradigm.xyz | bash\` and then running \`foundryup -C da2392e58bb8a7fefeba46b40c4df1afad8ccd22\` on a new terminal.
For more context, check the installation instructions in the book: https://book.getfoundry.sh/getting-started/installation.html.`
  )
  process.exit(1)
}

const version = execSync('forge --version').toString()


if (!version.includes('da239')) {
  console.warn(
    `Detected forge version ${version}. This may work, but if you run into any issues, please upgrade to forge da2392e58bb8a7fefeba46b40c4df1afad8ccd22.
Consider installing via \`curl -L https://foundry.paradigm.xyz | bash\` and then running \`foundryup -C da2392e58bb8a7fefeba46b40c4df1afad8ccd22\` on a new terminal.
For more context, check the installation instructions in the book: https://book.getfoundry.sh/getting-started/installation.html.`
  )
}
