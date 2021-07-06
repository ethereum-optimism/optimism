import { createLogger, format, transports } from 'winston'
import _ from 'lodash'

const colorizer = format.colorize()
const alignColorsAndTime = format.combine(
  format.colorize({
    message: true
  }),
  format.timestamp({
    format: 'YYYY-MM-DDTHH:mm:ssZ',
  }),
  format.metadata({ fillExcept: ['message', 'level', 'timestamp', 'label'] }),
  format.printf((info) => {
    const timestamp = colorizer.colorize(info.level, `[${info.level.toUpperCase()}] ${info.timestamp}:`)
    if (!_.isEmpty(info.metadata)) {
      const metadata = colorizer.colorize(info.level, info.metadata)
      return `${timestamp} ${info.message}\n${metadata}`
    }
    return `${timestamp} ${info.message}`
  })
)
const production = format.combine(
  format.timestamp({
    format: 'YYYY-MM-DDTHH:mm:ssZ',
  }),
  format.json(),
)

const logger = createLogger({
  level: (process.env.NODE_ENV === 'local') ? 'debug' : 'info',
  exitOnError: false,
  format: (process.env.NODE_ENV === 'local') ? alignColorsAndTime : production,
  transports: [ new transports.Console() ]
})

export default logger
