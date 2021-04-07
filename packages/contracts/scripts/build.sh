yarn run build:typescript &
yarn run build:contracts &
# avoid race condition between the 2 concurrent hardhat instances
sleep 2
yarn run build:contracts:ovm &

wait

yarn run build:copy:artifacts &
yarn run build:copy:artifacts:ovm &
yarn run build:copy:contracts &
wait
