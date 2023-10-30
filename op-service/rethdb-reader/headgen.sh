#!/bin/bash

set -e

# Generate rdb.h
cbindgen --crate rethdb-reader --output rdb.h -l C

# Process README.md to replace the content within the specified code block
awk '
  BEGIN { in_code_block=0; }
  /^```c/ { in_code_block=1; print; next; }
  /^```/ && in_code_block { in_code_block=0; while ((getline line < "rdb.h") > 0) print line; }
  !in_code_block { print; }
' README.md > README.tmp && mv README.tmp README.md

echo "Generated C header successfully"
