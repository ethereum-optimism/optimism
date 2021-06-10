#!/bin/sh
if [ -z "$1" ]; then
    exit 1
fi

if [[ $(ps -ef | grep -v grep | grep "$1" | wc -l) -eq 0 ]]; then
    exit 1
else
    exit 0
fi