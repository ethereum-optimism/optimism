// TODO add README

// How to run:
// 		go run ./op-chain-ops/cmd/celo-migrate-blocks -oldDB /path/to/oldDB -newDB /path/to/newDB [-batchSize 1000] [-memoryLimit 7500] [-clear-all] [-clear-nonAncients]
//
// This script will migrate block data from the old database to the new database
// The number of ancient records to migrate in one batch can be set using the -batchSize flag
// The default batch size is 1000
// You can set a memory limit in MB using the -memoryLimit flag. Defaults to 7500 MB. Make sure to set a limit that is less than your machine's available memory.
// Use -clear-all to start with a fresh new database
// Use -clear-nonAncients to keep migrated ancients, but not non-ancients
