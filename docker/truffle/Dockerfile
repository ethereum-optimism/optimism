# *******************************************************************************
# ******** pre-compiled omisego plasma contracts in a docker container  *********
# *******************************************************************************
FROM ajmay/truffle

RUN apt update -y \
    && apt install -y netcat jq python make g++

RUN git clone https://github.com/omgnetwork/plasma-contracts.git \
        && cd plasma-contracts/plasma_framework \
        && npm install \
        && npm install typescript
