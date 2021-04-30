rm -rf ./wallet/src/deployment &&
mkdir -p ./wallet/src/deployment &&
cp ./deployment/addresses.json ./wallet/src/deployment/addresses.json &&
cp -Rf ./artifacts ./wallet/src/deployment/artifacts &&
cp -Rf ./artifacts-ovm ./wallet/src/deployment/artifacts-ovm &&
wait