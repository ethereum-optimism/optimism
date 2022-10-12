# @eth-optimism/common-ts

## What is this?

`@eth-optimism/common-ts` contains useful tools for logging, metrics, and other Node stuff.

## BaseServiceV2

- BaseServiceV2 is the fastest way to create a typescript server.

- Options passed into optionsSpec in constructor are automatically turned into env variables that the service can consume

- Custom metrics can be registered in the constructor under metricSpecs

- To automatically run on a loop, create a main(){} function on your service. The constructor has options to control the polling interval.

- To run any code on startup, pass create an init(){} function on your service

### Creating APIs

- BaseServiceV2 supports really easy ways of building typesafe APIs that can be consumed by other typescript services

- Each route automatically comes with metrics

- To add cors config pass in cors option to constructor. Cors can be an array of whitelisted strings or regexes.

- To create a typesafe API use BaseServiceV2.createApi

```typescript
import { BaseServiceV2 } from 'eth-optimism/common-ts'
import { z } from 'zod'

const cors = [/^https:\/\/app\.optimism\.io/]

const trpcApi = BaseServiceV2.createAPI().query('user', {
  /**
   * Zod will validate input and also allow typescript to infer types
   */
  input: z.object({
    id: z.string(),
  }),
  // Resolve the endpoint here
  resolve: async (req) => {
    // this request will be typesafe
    return users.get(req.id)
  },
})

// exporting this type will allow you to consume your api on client
export type MyServiceAPI = typeof trpcApi

export class MyService extends BaseServiceV2<Options, Metrics, State> {
  constructor(params) {
    super({
      ...params,
      api: {
        cors,
        trpc: trpcApi,
      },
    })
  }
}
```

And then consume your API on client with typesafety!

```typescript
// we only import the types no runtime code from server is imported
import type { MyServiceAPI } from '../MyService'
import { createTRPCClient } from '@trpc/client'

const apiUri = process.env.API_URI ?? 'http://localhost:7300/api'

const trpcClient = createTRPCClient<MyServiceAPI>({
  url: apiUri,
})

// this is all typesafe with autocompl.etion
const user = await trpcClient.query('user', { id: '1' })``
```

see https://trpc.io/ version 9 for more info
