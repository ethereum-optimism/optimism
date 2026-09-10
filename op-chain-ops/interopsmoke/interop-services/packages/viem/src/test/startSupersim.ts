import { spawn } from 'child_process'
import fs from 'fs'
import path from 'path'

export type SupersimParameters = {
  l1Port?: number
  l2StartingPort?: number
}

export type SupersimReturnArg = Promise<() => void>

const READY_TEXT = 'supersim is ready'
const STD_OUT_PATH = path.resolve(__dirname, './stdout.txt')
const STD_ERR_PATH = path.resolve(__dirname, './stderr.txt')

export async function startSupersim(
  params?: SupersimParameters,
): SupersimReturnArg {
  const args = [] as string[]

  if (params?.l1Port) {
    args.push(`--l1.port ${params.l1Port}`)
  }

  if (params?.l2StartingPort) {
    args.push(` --l2.starting.port ${params.l2StartingPort}`)
  }

  fs.writeFileSync(STD_OUT_PATH, '')
  fs.writeFileSync(STD_ERR_PATH, '')

  const supersimProcess = spawn('supersim', args, { shell: true })

  const cleanup = () => {
    if (!supersimProcess.killed) {
      supersimProcess.kill()
    }
  }

  return new Promise((resolve, reject) => {
    supersimProcess.stdout.on('data', (data) => {
      const blob: string = data.toString()

      fs.appendFileSync(STD_OUT_PATH, blob.replace(/\\n/g, '\n'))

      if (blob.includes(READY_TEXT)) {
        resolve(cleanup)
      }
    })

    supersimProcess.stderr.on('data', (data) => {
      const blob: string = data.toString()
      fs.appendFileSync(STD_ERR_PATH, blob.replace(/\\n/g, '\n'))
      reject(blob)
    })
  })
}
