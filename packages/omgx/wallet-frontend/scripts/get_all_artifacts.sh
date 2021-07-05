ls -al
rm -rf src/deployment/artifacts &&
mkdir -p src/deployment/artifacts &&
rm -rf src/deployment/artifacts-ovm &&
mkdir -p src/deployment/artifacts-ovm &&

#no longer needed because the wallet pulls the addresses from the two http deployer services 
#cp -Rf ./deployment/local ../wallet-frontend/src/deployment &&
#cp -Rf ./deployment/rinkeby ../wallet-frontend/src/deployment &&

#these are the Base L1 contracts
cp -Rf ../../contracts/artifacts/contracts/optimistic-ethereum src/deployment/artifacts &&

#these are the Base L2 contracts
cp -Rf ../../contracts/artifacts-ovm/contracts/optimistic-ethereum src/deployment/artifacts-ovm &&

#these are the OMGX L1 contracts
cp -Rf ../contracts/artifacts/contracts src/deployment/artifacts &&

#these are the OMGX L2 contracts
cp -Rf ../contracts/artifacts-ovm/contracts src/deployment/artifacts-ovm &&

wait