# [services/airdrop.go](./airdrop.go)

Defines the Airdrop service which interacts with a Database and Metrics service to retrieve airdrop information.

- **Overview** 

The Airdrop struct encapsulates the Database and Metrics objects. It has one method GetAirdrop which is an HTTP handler to retrieve airdrop information for a specific address.

The NewAirdrop function is a constructor that creates an instance of Airdrop service by taking a Database and Metrics service as input.

The GetAirdrop method is an HTTP handler function that extracts an Ethereum address from the URL path parameters, retrieves the airdrop information for that address from the database, and responds with that information in JSON format. If there's a database error, it responds with a 500 Internal Server Error status and an error message. If the airdrop is not found for the given address, it responds with a 404 Not Found status.

