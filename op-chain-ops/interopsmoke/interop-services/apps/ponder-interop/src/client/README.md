# Ponder Interop Client

A TypeScript client for the ponder-interop API with runtime validation using Zod schemas.

## Features

- 🔒 **Runtime Validation** - All API responses are validated using Zod schemas
- 🚀 **Type Safety** - Full TypeScript support with types derived from schemas
- 🛡️ **Error Handling** - Comprehensive error handling with detailed validation messages
- 📦 **Easy to Use** - Simple constructor and intuitive method names

## Usage

```typescript
import { PonderInteropClient } from '@eth-optimism/ponder-interop/client'

const client = new PonderInteropClient('http://localhost:42069')

// All API calls are validated at runtime
try {
  const chains = await client.getChains()
  const pending = await client.getPendingMessages()

  console.log(`Found ${chains.length} chains`)
  console.log(`Pending messages: ${pending.length}`)
} catch (error) {
  // Will catch both network errors and validation errors
  console.error('API call failed:', error.message)
}
```

## Schema Validation

The client validates all API responses using Zod schemas. If the API returns unexpected data, you'll get a descriptive error:

```typescript
// Example validation error:
// "API response validation failed for /chains: {0.id: Expected number, received string}"
```

This helps catch API contract changes early and provides better debugging information.

## Available Methods

### Chains & System
- `getChains()` - Get all interoperable chains
- `getSchema()` - Get database schema information

### Messages
- `getMessageCount()` - Get message statistics
- `getPendingMessages()` - Get all pending messages
- `getPendingMessagesForAccount(account)` - Get pending messages for specific account

### Deposits
- `getDepositBalance(address)` - Aggregate deposit balance for an address
- `getDeposits()` - List depositors with aggregate balances
