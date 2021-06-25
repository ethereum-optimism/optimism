rm -rf ./wallet/src/deployment &&
mkdir -p ./wallet/src/deployment &&
cp -Rf ./deployment/local ./wallet/src/deployment &&
cp -Rf ./deployment/rinkeby ./wallet/src/deployment &&
cp -Rf ./artifacts ./wallet/src/deployment/artifacts &&
cp -Rf ./artifacts-ovm ./wallet/src/deployment/artifacts-ovm &&
cp -Rf ../../contracts/artifacts/* ./wallet/src/deployment/artifacts &&
cp -Rf ../../contracts/artifacts-ovm/* ./wallet/src/deployment/artifacts-ovm &&
wait