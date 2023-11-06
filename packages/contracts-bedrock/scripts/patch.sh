#!/bin/bash
set -euo pipefail
rm -Rf lib/openzeppelin-contracts-patched
mkdir -p lib/openzeppelin-contracts-patched
tar -C lib/openzeppelin-contracts -cf - . | tar -C lib/openzeppelin-contracts-patched -xvf -
patch -d lib/openzeppelin-contracts-patched -p3 < 'patches/@openzeppelin+contracts+4.7.3.patch'
