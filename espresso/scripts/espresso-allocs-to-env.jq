#!/usr/bin/env jq -S -r -f
to_entries | .[] | select(.value.name != null) | "\(.value.name)=\(.key)"
