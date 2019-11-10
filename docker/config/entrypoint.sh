#!/bin/sh

# Vault running in the container must listen on a different port.
export VAULT_ADDR="http://127.0.0.1:8900"

nohup vault server -dev -dev-root-token-id="totally-secure" \
  -log-level=debug -config=/home/vault/config/ -dev-listen-address="0.0.0.0:8900" &
VAULT_PID=$!

count=1
while [ "$count" -le 60 ]; do
  if vault status; then break; fi
  count=$((count+1))
  sleep 0.5
done

vault status

function configure_plugin {
	plugin_file="immutability-eth-plugin"

	echo "ADDING TO CATALOG: sys/plugins/catalog/secret/${plugin_file}"

	# just testing for now
	plugin_file="${plugin_file}"
	ls -latr /home/vault/plugins
	sha256sum=`cat /home/vault/plugins/SHA256SUMS | awk '{print $1}'`
	vault write sys/plugins/catalog/secret/${plugin_file} \
		  sha_256="$sha256sum" \
		  command="$plugin_file --ca-cert=/home/vault/ca/certs/ca.crt --client-cert=/home/vault/ca/certs/my-service.crt --client-key=/home/vault/ca/private/my-service.key"

	if [[ $? -eq 2 ]] ; then
	  echo "Vault Catalog update failed!"
	  exit 2
	fi

	echo "MOUNTING: ${plugin_file}"
	vault secrets enable -path=${plugin_file} -plugin-name=${plugin_file} plugin
	if [[ $? -eq 2 ]] ; then
	  echo "Failed to mount ${plugin_file} plugin for test!"
	  exit 2
	fi
}

configure_plugin

function test_banner {
    echo "************************************************************************************************************************************"
}

function test_plugin {
	test_banner
	echo "SMOKE TEST BASIC WALLET FUNCTIONALITY"
	test_banner
	/home/vault/scripts/smoke.wallet.sh
	test_banner
	echo "SMOKE TEST WHITELIST FUNCTIONALITY"
	test_banner
	/home/vault/scripts/smoke.whitelist.sh
	test_banner
	echo "SMOKE TEST BLACKLIST FUNCTIONALITY"
	test_banner
	/home/vault/scripts/smoke.blacklist.sh
	test_banner
	echo "SMOKE TEST ERC20 FUNCTIONALITY"
	test_banner
	/home/vault/scripts/smoke.erc20.sh
	test_banner
	echo "SMOKE TEST PLASMA FUNCTIONALITY"
	test_banner
	/home/vault/scripts/smoke.plasma.sh
}

test_plugin

# Log to STDOUT
vault audit enable file file_path=stdout

# Don't exit until vault dies
wait $VAULT_PID
