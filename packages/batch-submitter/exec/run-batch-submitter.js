#!/usr/bin/env node

console.log(process.env.npm_package_version);
process.exit(1)

const batchSubmitter = require("../dist/src/exec/run-batch-submitter")

batchSubmitter.run()
