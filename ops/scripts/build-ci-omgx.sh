# build in 2 steps
function build_images() {
    docker-compose build --parallel -- builder l2geth l1_chain
    docker-compose build --parallel -- deployer dtl batch_submitter relayer integration_tests
    docker-compose build --parallel -- omgx_deployer omgx_message-relayer-fast
    docker-compose build --parallel -- gas_oracle
}

function build_dependencies() {
    yarn
    yarn build
}

build_images &
build_dependencies &

wait
