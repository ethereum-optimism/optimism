#!/usr/bin/env zsh

cat - | jq -r '."outdated-chains" | sort_by(.name)[] | .name + "\n" + .diff.message + "\n" + (.diff.prestate | tostring) + "\n" + (.diff.latest | tostring)' | while read -r name; do
    read -r message
    read -r prestate
    read -r latest

    echo "\n=== $name ===\n$message\n"

    diff -u --label="$name-prestate" --label="$name-latest" <(echo "$prestate" | jq) <(echo "$latest" | jq)
done
