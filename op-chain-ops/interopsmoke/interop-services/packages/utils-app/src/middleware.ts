import { createMiddleware } from 'hono/factory'
import type { Logger } from 'pino'

/**
 * Middleware to log requests to the server.
 * @param log - the logger.
 * @returns Hono middleware function.
 */
export function requestLoggingMiddleware(log: Logger) {
  return createMiddleware(async (c, next) => {
    const { req } = c
    const now = Date.now()
    await next()
    const ms = Date.now() - now
    log.info(`${req.method} ${req.path} - ${ms}ms`)
  })
}
