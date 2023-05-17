# [services/l1/service.md](./service.go)

Internal service for interacting with the Layer 1 (L1) blockchain.

- **Overview**

The main struct here is the Service struct that contains a configuration object (ServiceConfig), context for cancellation, bridges (instances of different bridge implementations), a batch scanner (for scanning the state commitment chain), and a lot of other related components.

The Service struct also includes methods for updating the service with a new Ethereum block header (Update), catching up to the current block (catchUp), and looping over the service to constantly update the state of the Ethereum chain (loop).

The package also defines HTTP handlers for fetching the status of the indexer and getting deposits for a specific address.

