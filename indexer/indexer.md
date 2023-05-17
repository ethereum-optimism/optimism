# [indexer.go](./cmd/indexer/main.go)

The main entrypoint to the indexer application.

- **Overview**

The service is designed to index Ethereum blockchain data from L1 and L2 nodes, store that data in a database, and serve it over a REST API. It also serves health checks and can be configured to provide metrics.  Here's a summary of it's core parts:

- **Main Function** 

This is the main entry point of the service. It creates a new configuration, initializes an indexer, starts it, and then waits indefinitely. If an error occurs during initialization or start, the service will stop and the error will be returned.

- **Indexer Struct** 

The Indexer struct encapsulates the main components of the service: Ethereum clients for L1 and L2, L1 and L2 indexing services, a router for handling HTTP requests, and other resources like the database and metrics.

- **NewIndexer Function** 

This function initializes a new Indexer. It sets up logging, connects to the L1 and L2 Ethereum clients, initializes the metrics server if enabled, connects to the database, creates address manager services based on configuration, and sets up L1 and L2 indexing services.

- **Serve Method (deprecated)** 

This method sets up a REST API server. It allows CORS from all origins, defines various API endpoints for L1, L2 status, deposits, withdrawals, and airdrops, and sets up a health check endpoint. Then it starts an HTTP server on a specified host and port.

[mux](https://github.com/gorilla/mux) is used to create an api.  This is no longer maintained in favor of a TRPC based typescript API.

- **Start Method**

This method starts the indexing services for L1 and L2 (unless disabled by configuration), and calls the Serve method to start the REST server.

- **Stop Method** 

This method stops the indexing services, closes the database connection, and shuts down the HTTP server.

- **dialEthClientWithTimeout Function** 

This helper function is used to dial an Ethereum client at a given URL with a timeout.


- **See also:**

- Implementation in [indexer.go](./indexer.go)
- [go-ethereum](github.com/ethereum/go-ethereum)
- [go-ethereum common utils](github.com/ethereum/go-ethereum/common)
- [go-ethereum ethclient docs](github.com/ethereum/go-ethereum/ethclient)
- [go-ethereum rpc docs](github.com/ethereum/go-ethereum/rpc)
- [internal services](./services/README.md)
- [internal metrics docs](./metrics/README.md)
- [internal (deprecated) server docs](./server/README.md)
- [internal db docs](./db/README.md)

