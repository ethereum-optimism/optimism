import colors from 'colors/safe'

export class Logger {
  constructor(public namespace: string) {}

  public info(message: string): void {
    console.log(`${colors.cyan(`[${this.namespace}] [INFO]`)}: ${message}`)
  }

  public status(message: string): void {
    console.log(`${colors.magenta(`[${this.namespace}] [STATUS]`)}: ${message}`)
  }

  public interesting(message: string): void {
    console.log(`${colors.yellow(`[${this.namespace}] [INFO]`)}: ${message}`)
  }

  public success(message: string): void {
    console.log(`${colors.green(`[${this.namespace}] [SUCCESS]`)}: ${message}`)
  }

  public error(message: string): void {
    console.log(`${colors.red(`[${this.namespace}] [ERROR]`)}: ${message}`)
  }
}
