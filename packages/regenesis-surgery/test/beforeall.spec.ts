import { env } from './setup'

before('initializing test environment', async () => {
  await env.init()
})
