yarn run build:typescript &
yarn run build:contracts:ovm &
yarn run build:contracts &

wait

yarn run build:copy:artifacts &
yarn run build:copy:artifacts:ovm &
yarn run build:copy:contracts &
wait
