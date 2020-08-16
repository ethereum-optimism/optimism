#!/bin/bash

vault secrets disable immutability-eth-plugin
vault delete sys/plugins/catalog/secret/immutability-eth-plugin
cd ..
go clean && go build
mv immutability-eth-plugin $HOME/etc/vault.d/vault_plugins/immutability-eth-plugin
export SHA256=$(shasum -a 256 "$HOME/etc/vault.d/vault_plugins/immutability-eth-plugin" | cut -d' ' -f1)
vault write sys/plugins/catalog/secret/immutability-eth-plugin \
      sha_256="${SHA256}" \
      command="immutability-eth-plugin --ca-cert=$HOME/etc/vault.d/root.crt --client-cert=$HOME/etc/vault.d/vault.crt --client-key=$HOME/etc/vault.d/vault.key"
vault secrets enable -path=immutability-eth-plugin -plugin-name=immutability-eth-plugin plugin
