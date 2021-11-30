export const writeStderr = (msg: string) => {
  process.stderr.write(`${msg}\n`)
}
