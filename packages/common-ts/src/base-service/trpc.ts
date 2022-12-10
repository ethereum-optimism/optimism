import { BuildProcedure, CreateRouterInner, initTRPC } from '@trpc/server'
import superjson from 'superjson'
import { z } from 'zod'
import * as trpcExpress from '@trpc/server/adapters/express'

import { Logger } from '../common'
import { MetricsV2 } from './base-service-v2'

/**
 * Extend this class to create a new trpc server
 *
 * @see https://trpc.io/docs/v10/quick-start
 * @example
 * class TrpcApi extends Trpc<Metrics> {
 *   public readonly metadataController = this.trpc.procedure
 *     .input(
 *       z.object({
 *         address: ZodHelpers.ethereumAddress,
 *       }),
 *     )
 *     .query(async (req) => {
 *       return this.keyValueStorage.get(createMetadataKey(req.input.address))
 *     })
 *
 *  routes = this.router({
 *    metadata: this.trpc.procedure
 *      .input(
 *       z.object({
 *        address: this.z.string().refine(ethers.utils.isAddress),
 *      }),
 *    )
 *   .query(async (req) => {
 *      return this.keyValueStorage.get(createMetadataKey(req.input.address))
 *    }),
 *  })
 * }
 *
 * @important - You can then create a typesafe client for vanillajs, react, and more
 * @see https://trpc.io/docs/v10/client
 * @important - See vanilla js examples in trpc documentation if not using react
 * @example - react example
 * // import only the type so you don't have to import any actual code
 * import type {Api} from '../backend/api'
 * import { createTRPCReact, httpBatchLink } from '@trpc/react'
 *
 * // the client can infer the api types from the routes
 * export const apiClient = createTRPCReact<Api['routes']>()
 *
 * // you can then use it in a typesafe way in your client code
 * apiClient.metadata.useQuery({address: userAddress})
 */
export abstract class TrpcApi<TMetrics extends MetricsV2> {
  /**
   * Turns an trpc server into express middleware
   *
   * @see https://trpc.io/docs/v10/express
   * @example
   * app.use('/', api.createExpressMiddleware())
   */
  public readonly createExpressMiddleware = () =>
    trpcExpress.createExpressMiddleware({
      router: this.routes,
    })

  /**
   * quality of life property.  Zod is used to validate input in a
   * typesafe way.
   * Can be used by `import {z} from 'zod' as well.
   *
   * @see https://github.com/colinhacks/zod
   */
  protected readonly z = z
  /**
   * @see https://trpc.io/docs/v10/router
   */
  protected readonly router = this.trpc.router
  /**
   * @see https://trpc.io/docs/v10/merging-routers
   */
  protected readonly mergeRouters = this.trpc.mergeRouters
  /**
   * @see https://trpc.io/docs/v10/procedures
   **/
  protected readonly procedure = this.trpc.procedure
  /**
   * @see https://trpc.io/docs/v10/middlewares
   */
  protected readonly middleware = this.trpc.middleware

  /**
   * Routes are objects of route names to route handlers
   * You can also recursively set other routes as route
   * handlers to create nested routes
   *
   * @see https://trpc.io/docs/v10/procedures
   * @example typescript
   *   public readonly metadataController = this.trpc.procedure
   *     .input(
   *       z.object({
   *         address: ZodHelpers.ethereumAddress,
   *       }),
   *     )
   *     .query(async (req) => {
   *       return this.keyValueStorage.get(createMetadataKey(req.input.address))
   *     })
   *
   *  routes = this.router({
   *     metadata: this.metadataController,
   *     nestedroute: this.router({
   *        nestedRoute: this.nestedRouteHandler
   *     })
   *  })
   *
   */
  public abstract readonly routes:
    | BuildProcedure<any, any, any>
    | CreateRouterInner<any, any>

  /**
   * Class is not meant to be instantiated directly
   * but rather extended
   */
  constructor(
    protected metrics: TMetrics,
    protected logger: Logger,
    private readonly trpc = initTRPC.create({
      /**
       * @see https://trpc.io/docs/v10/data-transformers
       */
      transformer: superjson,
    })
  ) {}
}
