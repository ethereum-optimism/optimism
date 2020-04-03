#!/bin/sh

set -ex

function configure_plugin {
	PLUGIN_FILE="immutability-eth-plugin"
	SHA256SUM="9cafb169cfa7be7574fa0986a15de0119785855b3978ab632f226f4c3509c385"
	echo "ADDING TO CATALOG: sys/plugins/catalog/secret/$PLUGIN_FILE"

	vault write sys/plugins/catalog/secret/$PLUGIN_FILE \
		  sha_256="$SHA256SUM" \
		  command="$PLUGIN_FILE --ca-cert=/etc/tls/ca/tls.crt --client-cert=/etc/tls/service/tls.crt --client-key=/etc/tls/service/tls.key"

	if [[ $? -eq 2 ]] ; then
	  echo "Vault Catalog update failed!"
	  exit 2
	fi

	echo "MOUNTING: $PLUGIN_FILE"
	vault secrets enable -path=$PLUGIN_FILE -plugin-name=$PLUGIN_FILE -description="OmiseGo Plasma Authority Wallet" plugin
	if [[ $? -eq 2 ]] ; then
	  echo "Failed to mount $PLUGIN_FILE plugin for test!"
	  exit 2
	fi
}

configure_plugin