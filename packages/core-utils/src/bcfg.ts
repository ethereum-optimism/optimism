export interface Bcfg {
  load: (options: { env?: boolean; argv?: boolean }) => void
  str: (name: string, defaultValue?: string) => string
  uint: (name: string, defaultValue?: number) => number
  bool: (name: string, defaultValue?: boolean) => boolean
  ufloat: (name: string, defaultValue?: number) => number
  has: (name: string) => boolean
}
