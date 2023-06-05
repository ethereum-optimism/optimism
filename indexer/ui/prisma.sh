#!/usr/bin/env bash

# This script updates the prisma schema
#
SCRIPT_DIR=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )

DATABASE_URL=${DATABASE_URL:-postgresql://db_username:db_password@localhost:5434/db_name}
PRISMA_FILE="$SCRIPT_DIR/schema.prisma"
TEMP_FILE="$SCRIPT_DIR/temp-schema.prisma"

function update_prisma() {
    echo "Updating Prisma Schema..."
    npx prisma db pull --url $DATABASE_URL --schema $PRISMA_FILE
    echo "Update completed."
}

function check_prisma() {
    echo "Checking Prisma Schema..."
    cp $PRISMA_FILE $TEMP_FILE
    npx prisma db pull --url $DATABASE_URL --schema $TEMP_FILE
    diff $PRISMA_FILE $TEMP_FILE > /dev/null
    if [ $? -eq 0 ]; then
        echo "Prisma Schema is up-to-date."
        rm $TEMP_FILE
    else
        echo "Prisma Schema is not up-to-date."
        rm $TEMP_FILE
        return 1
    fi
}

if [ "$1" == "--check" ]; then
    check_prisma
else
    update_prisma
fi
