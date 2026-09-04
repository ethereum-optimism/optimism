import { setupTicTacToe } from '@/test/setupTicTacToe.js'

import { startSupersim } from './startSupersim.js'

async function main() {
  let shutdown

  try {
    shutdown = await startSupersim()
    await setupTicTacToe()
  } catch (e) {
    process.exit(1)
  }

  return () => {
    shutdown()
  }
}

export default main
