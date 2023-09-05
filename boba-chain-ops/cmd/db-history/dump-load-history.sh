#!/bin/bash

TABLES="Header \
	BlockBody \
	BlockTransaction \
	Receipt \
	HeaderNumber \
	CanonicalHeader \
	HeadersTotalDifficulty \
	BlockTransactionLookup \
	SyncStage \
	LastBlock \
	LastHeader "

USAGE="usage: $0 <dump|load> <db_path> <dump_path>"

if [ $1 == '-h' ] ; then
  echo $USAGE
  exit 0
fi

if [ $# -ne 3 ] ; then
  echo $USAGE
  exit 1
fi

if [ ! -d $2 ] ; then
  echo "<db_path> is not a directory"
  echo $USAGE
  exit 1
fi

mkdir -p $3  # Create dump dir if needed. Note that this is not done for db_path
if [ ! -d $3 ] ; then
  echo "<dump_path> is not a directory"
  echo $USAGE
  exit 1
fi

MDBX_DUMP=`PATH=.:$PATH which mdbx_dump`
if [ $? -ne 0 ] ; then
  echo "Did not find mdbx_dump (from erigon db-tools) in current dir or \$PATH"
  exit 2
fi

MDBX_LOAD=`PATH=.:$PATH which mdbx_load`
if [ $? -ne 0 ] ; then
  echo "Did not find mdbx_load (from erigon db-tools) in current dir or \$PATH"
  exit 2
fi

set -e

if [ $1 == 'dump' ] ; then
  echo "Dumping tables from $2 to $3"
  for t in $TABLES ; do
    echo $t
    $MDBX_DUMP -s $t -f $3/$t.dump $2/chaindata
  done
elif [ $1 == 'load' ] ; then
  echo "Loading tables from $3 to $2 (will purge + overwrite)"
  for t in $TABLES ; do
    echo $t
    $MDBX_LOAD -p -s $t -f $3/$t.dump $2/chaindata
  done
else
  echo $USAGE
  exit 1
fi

echo "Done"
