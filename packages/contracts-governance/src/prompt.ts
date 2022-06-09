import readline from 'readline'

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
})

export const prompt = (msg: string) =>
  new Promise<void>((resolve, reject) =>
    rl.question(`${msg} [y/n]: `, (confirmation) => {
      if (confirmation !== 'y') {
        reject('Aborted!')
      }

      resolve()
    })
  )
