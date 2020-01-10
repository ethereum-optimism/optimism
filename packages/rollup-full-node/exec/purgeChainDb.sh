#!/bin/bash

read -r -p "Are you sure you want to irreversibly delete the L2 chain DB? [y/N] " input
case "$input" in
    [yY][eE][sS]|[yY])
        echo "deleting chain DB..."
        parity db kill --chain $(dirname $0)/../config/parity/local-chain-config.json
        ;;
    *)
        echo "Phew! Dodged a bullet there."
        ;;
esac
