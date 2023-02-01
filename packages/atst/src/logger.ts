import util from 'util'

import ora from 'ora'
import pc from 'picocolors'

const format = (args: any[]) => {
  return util
    .format(...args)
    .split('\n')
    .join('\n')
}

export const success = (...args: Array<any>) => {
  console.log(pc.green(format(args)))
}

export const info = (...args: Array<any>) => {
  console.info(pc.blue(format(args)))
}

export const log = (...args: Array<any>) => {
  console.log(pc.white(format(args)))
}

export const warn = (...args: Array<any>) => {
  console.warn(pc.yellow(format(args)))
}

export const error = (...args: Array<any>) => {
  console.error(pc.red(format(args)))
}

export const spinner = () => {
  return ora({
    color: 'gray',
    spinner: 'dots8Bit',
  })
}
