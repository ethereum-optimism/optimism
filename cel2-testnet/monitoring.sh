#!/bin/bash
set -euo pipefail

function send_message {
	local msg="$1"
	curl https://slack.com/api/chat.postMessage \
		-X POST -H 'Content-type: application/json' \
		-H "Authorization: Bearer $MONITORING_SLACK_SECRET" \
		--data '{"channel": "'"$MONITORING_SLACK_CHANNEL"'", "text": "'"$msg"'"}'
}

SCRIPT_DIR=$(readlink -f $(dirname $0))
cd $SCRIPT_DIR
source .envrc


### Balance check ###
balance_text=$($SCRIPT_DIR/balances.sh) \
	|| send_message "*Cel2 Sepolia wallet balances*\n\`\`\`$balance_text\`\`\`"


### Alchemy API check ###
ALCHEMY_HTTP_CODE=$(
	curl $L1_RPC -sS -o /dev/null -w "%{http_code}" \
	-X POST \
	-H "Content-Type: application/json" \
	--data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}')

# 403 means out of quota, see https://docs.alchemy.com/reference/error-reference
if [ $ALCHEMY_HTTP_CODE = 403 ]
then
	send_message "*WARNING*: The cel2 testnet is out of alchemy quota, leading to a stall of the network."
fi


### End notice to verify monitoring works when no warnings are generated ###
echo Monitoring finished at $(date)
