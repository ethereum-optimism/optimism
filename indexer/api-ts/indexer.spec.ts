import { test, expect } from 'vitest'
import { depositEndpoint, withdrawalEndoint } from './indexer.ts'

test(depositEndpoint.name, () => {
  expect(depositEndpoint({ baseUrl: 'http://localhost:8080/api/v0', address: '0x1234', cursor: '0x1235', limit: 10 })).toMatchInlineSnapshot('"http://localhost:8080/api/v0/deposits/0x1234?cursor=0x1235&limit=10"')
  expect(depositEndpoint({ baseUrl: 'http://localhost:8080/api/v0', address: '0x1234' })).toMatchInlineSnapshot('"http://localhost:8080/api/v0/deposits/0x1234"')
})

test(withdrawalEndoint.name, () => {
  expect(withdrawalEndoint({ baseUrl: 'http://localhost:8080/api/v0', address: '0x1234', cursor: '0x1235', limit: 10 })).toMatchInlineSnapshot('"http://localhost:8080/api/v0/withdrawals/0x1234?cursor=0x1235&limit=10"')
  expect(withdrawalEndoint({ baseUrl: 'http://localhost:8080/api/v0', address: '0x1234' })).toMatchInlineSnapshot('"http://localhost:8080/api/v0/withdrawals/0x1234"')
})
