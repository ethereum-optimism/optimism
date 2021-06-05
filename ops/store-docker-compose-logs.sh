#!/bin/bash

set -e

# Create artifacts folder for logs
artifacts_folder="./artifacts/$PKGS"
mkdir -p $artifacts_folder

docker-compose logs \
    --no-color > $artifacts_folder/process.log # Send all process logs to process.log

# Send all process logs to artifacts folder w/ service name as filename
cat $artifacts_folder/process.log | grep -e "|" | \
    # Delimiter based on | which docker-compose uses in streamed logs
    awk '{
        idx = index($0, "| ");                  # Assign index of delimiter between service and log
        service = substr($0, 0, idx);           # Assign variable for service
        gsub("[^a-zA-Z0-9_]", "", service);     # Substitute any unnatural characters (e.g., spaces)
        gsub("$", ".log", service);             # Add .log to end of variable (e.g., l1_chain.log)
        outputfile = sprintf (service);         # Assign output file name to variable
        print substr($0, idx + 2) > outputfile; # Send corresponding log to corresponding log file
    }'