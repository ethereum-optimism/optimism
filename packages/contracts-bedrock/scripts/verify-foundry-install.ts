import { sync as commandExistsSync } from 'command-exists'

if (!commandExistsSync('forge')) {
  console.error(
    'Command failed. Is Foundry not installed? Consider installing via `curl -L https://foundry.paradigm.xyz | bash` and then running `foundryup` on a new terminal. For more context, check the installation instructions in the book: https://book.getfoundry.sh/getting-started/installation.html.'
  )
  process.exit(1)
}
