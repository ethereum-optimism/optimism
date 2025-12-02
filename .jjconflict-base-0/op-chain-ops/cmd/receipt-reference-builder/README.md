# Receipt Reference Tool

Receipt Reference Tool is a data-pulling tool for operational use by Superchain operators of chains which have Post-Bedrock-Pre-Canyon activity.

## Data Collection

### Pull
The `pull` subcommand manages a collection of workers to request blocks from an RPC endpoint, and then checks each block for deposit transactions. Those transactions are built up into an aggregate data structure and written.

### Merge
The `merge` subcommand targets an array of files, confirms that there is no gap in the processed block ranges, and then merges the aggregates into a single file.

### Convert
The `convert` subcommand targets a single file and writes it as a new file in the requested format.

### Print
`print` is a debug subcommand to read in a file and print it to screen.

## Data Spec
The output data of this tool is an "aggregate". Each aggregate contains the following attributes
- Start Block, End Block
- Chain ID
- Results Map:
  - Key of BlockNumber
  - Value of Nonces as a slice

Transaction Nonces are inserted to the value slice in the order they appear in the block. Transaction Nonces are only included if they are related to a user deposit.
Blocks which contain no deposit transactions have no key in the data.

Users of this data can easily find if the data is appropriate for their network (using ChainID), covers a given block (using Start and End), and provides the nonces for user deposits.

## Best Practices
This tool is designed with a static range of blocks in mind, the size of which is about 10 Million blocks. In order to get such a large body of data in one place, this tool is built for parallel execution and retries.

To maximize parallel efficiency, a higher number of `-workers` can utilize more RPC requests per second. Additionally `-batch-size` can be increased to group more RPC requests together per network exchange. I am using 5 workers with 100 requests per batch.

To avoid wasteful abandon of work already done, errors which are encountered by workers are noted, but do not stop the aggregation process. Jobs which fail are reinserted into the work queue with no maximum retry, and workers back off when encountering failures. This is all to allow an RPC endpoint to become temporarily unavailable while letting aggregation stay persistent.

Even at high speed, collecting this much data can take several hours. You may benefit from planning a collection of smaller-sized runs, merging them with the `merge` subcommand as they become available.
