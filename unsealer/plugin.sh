#!/bin/sh

set -ex

function configure_plugin {
	plugin_file="immutability-eth-plugin"

	echo "ADDING TO CATALOG: sys/plugins/catalog/secret/${plugin_file}"

	# just testing for now
	plugin_file="${plugin_file}"
	sha256sum=`cat SHA256SUMS | awk '{print $1}'`
	vault write sys/plugins/catalog/secret/${plugin_file} \
		  sha_256="$sha256sum" \
		  command="$plugin_file --ca-cert=/etc/tls/ca/tls.crt --client-cert=/etc/tls/service/tls.crt --client-key=/etc/tls/service/tls.key"

	if [[ $? -eq 2 ]] ; then
	  echo "Vault Catalog update failed!"
	  exit 2
	fi

	echo "MOUNTING: ${plugin_file}"
	vault secrets enable -path=${plugin_file} -plugin-name=${plugin_file} -description="OmiseGo Plasma Authority Wallet" plugin
	if [[ $? -eq 2 ]] ; then
	  echo "Failed to mount ${plugin_file} plugin for test!"
	  exit 2
	fi
}

configure_plugin