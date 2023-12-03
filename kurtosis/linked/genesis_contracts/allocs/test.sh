#cat allocs.json | jq -r '. | to_entries | map({
#  key: .key,
#  value: (.value.storage |= to_entries | map({
#    key: (.key | ltrimstr("0x") | lpad(64; "0") | ("0x" + .)),
#    value: (.value | ltrimstr("0x") | lpad(64; "0") | ("0x" + .))
#  }) | from_entries) | from_entries
#})'


cat allocs.json |
