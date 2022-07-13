---
'@eth-optimism/ci-builder': minor
---

Fix unbound variable in check_changed script

This now uses -z to check if a variable is unbound instead of -n.
This should fix the error when the script is being ran on develop.
