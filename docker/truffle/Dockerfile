# *******************************************************************************
# ******** pre-compiled omisego plasma contracts in a docker container  *********
# *******************************************************************************
FROM ajmay/truffle

RUN apt update -y \
    && apt install -y netcat jq

RUN git clone https://github.com/omisego/plasma-contracts.git \
        && cd plasma-contracts/plasma_framework \
        && npm install \
        && truffle compile \
        && truffle tests
