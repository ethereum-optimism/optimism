import { ValidatorSpec, Spec } from 'envalid'

import { LogLevel } from '../common/logger'
import { validators } from './validators'

/**
 * Options for a service.
 */
export type Options = {
  [key: string]: any
}

/**
 * Specification for options.
 */
export type OptionsSpec<TOptions extends Options> = {
  [P in keyof Required<TOptions>]: {
    validator: (spec?: Spec<TOptions[P]>) => ValidatorSpec<TOptions[P]>
    desc: string
    default?: TOptions[P]
    public?: boolean
  }
}

/**
 * Standard options shared by all services.
 */
export type StandardOptions = {
  loopIntervalMs?: number
  port?: number
  hostname?: string
  logLevel?: LogLevel
  useEnv?: boolean
  useArgv?: boolean
}

/**
 * Specification for standard options.
 */
export const stdOptionsSpec: OptionsSpec<StandardOptions> = {
  loopIntervalMs: {
    validator: validators.num,
    desc: 'Loop interval in milliseconds, only applies if service is set to loop',
    default: 0,
    public: true,
  },
  port: {
    validator: validators.num,
    desc: 'Port for the app server',
    default: 7300,
    public: true,
  },
  hostname: {
    validator: validators.str,
    desc: 'Hostname for the app server',
    default: '0.0.0.0',
    public: true,
  },
  logLevel: {
    validator: validators.logLevel,
    desc: 'Log level',
    default: 'debug',
    public: true,
  },
  useEnv: {
    validator: validators.bool,
    desc: 'For programmatic use, whether to use environment variables',
    default: true,
    public: true,
  },
  useArgv: {
    validator: validators.bool,
    desc: 'For programmatic use, whether to use command line arguments',
    default: true,
    public: true,
  },
}

/**
 * Gets the list of public option names from an options specification.
 *
 * @param optionsSpec Options specification.
 * @returns List of public option names.
 */
export const getPublicOptions = (
  optionsSpec: OptionsSpec<Options>
): string[] => {
  return Object.keys(optionsSpec).filter((key) => {
    return optionsSpec[key].public
  })
}
