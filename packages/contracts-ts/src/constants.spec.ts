import { test, expect } from 'vitest'
import { addresses } from './constants'
import { readFileSync } from 'fs'
import { join } from 'path'

const jsonAddresses = JSON.parse(
  readFileSync(join(__dirname, '../addresses.json'), 'utf8')
)

test('should have generated addresses', () => {
  expect(addresses).toEqual(jsonAddresses)
})
