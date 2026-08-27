# Ponder Interop Test Suite

This directory contains comprehensive test coverage for the ponder-interop application, covering both database indexing functionality and REST API endpoints.

## Test Structure

```
__tests__/
├── setup.ts                    # Test environment setup and global mocks
├── types.d.ts                  # TypeScript declarations for test helpers
├── indexing/                   # Database indexing tests
│   ├── l2-to-l2-cdm.test.ts   # L2 to L2 cross-domain message indexing
│   ├── gas-tank.test.ts        # Gas tank event indexing
│   ├── promises.test.ts        # Promise contract event indexing
│   └── handlers/               # Extracted handler functions for testing
│       ├── l2-to-l2-cdm-handlers.ts
│       ├── gas-tank-handlers.ts
│       └── promise-handlers.ts
├── api/                        # REST API tests
│   └── simple-api.test.ts     # Simplified API tests
├── client/                     # API client tests
│   └── client.test.ts         # PonderInteropClient tests
├── utils/                      # Utility function tests
│   └── hashMessageIdentifier.test.ts
└── integration/                # Integration tests
    └── integration.test.ts     # End-to-end workflow tests
```

## Test Categories

### 1. Indexing Tests (`indexing/`)

Tests the core indexing functionality that processes blockchain events and stores them in the database.

#### L2 to L2 Cross-Domain Messages (`l2-to-l2-cdm.test.ts`)
- **SentMessage Event Handler**: Tests indexing of sent cross-domain messages
- **RelayedMessage Event Handler**: Tests both v1 and v2 variants of relayed messages
- **Message Identifier Hashing**: Verifies correct computation of message identifier hashes
- **Cross-Domain Message Hashing**: Validates message hash generation
- **Log Payload Encoding**: Tests proper encoding of event log data
- **Error Handling**: Database errors, malformed events

#### Gas Tank (`gas-tank.test.ts`)
- **Deposit Events**: Gas provider deposits and balance updates
- **Withdrawal Events**: Initiation and finalization of withdrawals
- **Flagged Messages**: Message flagging functionality
- **Claimed Messages**: Gas claim processing and balance deduction
- **Gas Receipts**: Relayed message gas receipt tracking
- **Conflict Resolution**: Database upserts and conflict handling

#### Promises (`promises.test.ts`)
- **Promise Creation**: Event indexing and global ID computation
- **Promise Resolution**: Status updates and return data storage
- **Promise Rejection**: Error data handling and status updates
- **Lifecycle Management**: Complete promise workflow from creation to resolution
- **Cross-Chain Scenarios**: Multi-chain promise handling
- **Global ID Computation**: Deterministic cross-chain promise identification

### 2. API Tests (`api/`)

Tests the REST API endpoints that serve indexed data.

#### Endpoints Tested
- `GET /chains` - List of interoperable chains
- `GET /schema` - Database schema information
- `GET /messages/count` - Message count statistics
- `GET /messages/pending` - Pending messages list
- `GET /messages/:account/pending` - Account-specific pending messages
- `GET /promises` - All promises
- `GET /promises/pending` - Pending promises
- `GET /promises/resolved` - Resolved promises
- `GET /promises/rejected` - Rejected promises

#### Test Coverage
- **Response Validation**: Schema validation and data format verification
- **Error Handling**: Database errors, invalid parameters, HTTP errors
- **BigInt Conversion**: Proper conversion of BigInt values to numbers
- **Pagination**: Limit enforcement and query optimization
- **Content Types**: JSON response headers and formatting

### 3. Client Tests (`client/`)

Tests the API client library that applications use to interact with the ponder-interop API.

#### PonderInteropClient Features
- **Constructor**: Base URL handling and trailing slash removal
- **Factory Function**: Client creation via factory method
- **HTTP Methods**: All API endpoint wrappers
- **Error Handling**: Network errors, HTTP errors, validation errors
- **Response Validation**: Zod schema validation for all responses
- **Type Safety**: Full TypeScript support with proper typing

#### Test Scenarios
- **Success Cases**: Proper data fetching and response parsing
- **Network Errors**: Timeout handling and connection failures
- **Validation Errors**: Schema mismatch and malformed responses
- **Promise Statistics**: Aggregated promise status calculations

### 4. Utility Tests (`utils/`)

Tests utility functions used throughout the application.

#### hashMessageIdentifier
- **Basic Functionality**: Correct hash generation and format validation
- **Deterministic Behavior**: Consistent hashing for identical inputs
- **Field Sensitivity**: Different hashes for different field values
- **Edge Cases**: Zero values, maximum values, boundary conditions
- **Implementation Correctness**: Matches manual encoding and hashing
- **Parameter Order**: Correct ABI encoding parameter order
- **Real-world Scenarios**: Mainnet and L2 chain identifier handling
- **Collision Resistance**: Unique hashes for similar inputs

### 5. Integration Tests (`integration/`)

End-to-end tests that verify complete workflows across all components.

#### Test Scenarios
- **Message Flow**: Event → Indexing → API → Client response
- **Promise Lifecycle**: Creation → Resolution → API query
- **Gas Tank Operations**: Deposit → Claim → Balance updates
- **Utility Integration**: Hash computation consistency
- **Error Propagation**: Error handling across all layers

## Running Tests

### All Tests
```bash
pnpm test              # Run all tests in watch mode
pnpm test:run          # Run all tests once
pnpm test:coverage     # Run with coverage report
```

### Specific Test Categories
```bash
pnpm test:indexing     # Database indexing tests only
pnpm test:api          # REST API tests only
pnpm test:client       # API client tests only
pnpm test:utils        # Utility function tests only
pnpm test:integration  # Integration tests only
```

### Development Workflow
```bash
pnpm test:watch        # Watch mode for development
```

## Test Environment

### Setup (`setup.ts`)
- **Environment Variables**: Mock configuration for tests
- **Global Helpers**: Mock functions for events, transactions, and blocks
- **Console Suppression**: Reduces noise during test execution

### Type Declarations (`types.d.ts`)
- **Global Functions**: TypeScript declarations for test helpers
- **Mock Types**: Proper typing for mock objects

### Mock Strategy
- **Minimal Mocking**: Only mock external dependencies (ponder modules)
- **Real Logic**: Test actual implementation code where possible
- **Extracted Handlers**: Testable functions extracted from main indexing logic

## Coverage Goals

The test suite aims for comprehensive coverage across:

1. **Functional Coverage**: All public functions and methods
2. **Branch Coverage**: All conditional logic paths
3. **Error Coverage**: All error handling scenarios
4. **Integration Coverage**: Complete workflows and data flow
5. **Edge Case Coverage**: Boundary conditions and edge cases

## Test Data

### Mock Data Patterns
- **Realistic Values**: Use realistic blockchain data (addresses, hashes, timestamps)
- **Edge Cases**: Test with zero values, maximum values, and boundary conditions
- **Type Safety**: Ensure all mock data matches expected TypeScript types
- **Consistency**: Use consistent mock data patterns across all tests

### Mock Helpers
- `mockEventLog()`: Creates mock blockchain event logs
- `mockTransaction()`: Creates mock transaction data
- `mockBlock()`: Creates mock block data

## Best Practices

1. **Descriptive Test Names**: Clear descriptions of what each test validates
2. **Arrange-Act-Assert**: Clear test structure with setup, execution, and verification
3. **Error Testing**: Comprehensive error handling and edge case coverage
4. **Mock Isolation**: Tests don't depend on external systems or real blockchain data
5. **Type Safety**: Full TypeScript coverage including test code
6. **Documentation**: Clear comments explaining complex test logic

## Contributing

When adding new functionality:

1. **Add Tests First**: Write tests for new features during development
2. **Cover All Paths**: Ensure both success and error paths are tested
3. **Integration Tests**: Add integration tests for new workflows
4. **Update Documentation**: Update this README for new test categories
5. **Run Full Suite**: Ensure all tests pass before submitting changes

## Debugging Tests

### Common Issues
- **Mock Setup**: Ensure mocks are properly configured for the tested functionality
- **Type Errors**: Check TypeScript declarations in `types.d.ts`
- **Import Paths**: Verify correct import paths for test modules
- **Async Handling**: Ensure proper `await` usage for async operations

### Debug Commands
```bash
# Run specific test file
npx vitest run __tests__/indexing/l2-to-l2-cdm.test.ts

# Run with debug output
npx vitest run --reporter=verbose

# Run single test case
npx vitest run -t "should correctly index a sent message"
```