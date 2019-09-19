export interface Logger {
  debug: (...args: any[]) => any
  info: (...args: any[]) => any
  error: (...args: any[]) => any
}
