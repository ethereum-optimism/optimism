FROM node:16-alpine
ARG version=v2.56.1

RUN apk add --no-cache git curl python3 bash jq

WORKDIR /synthetix
RUN git clone -b $version https://github.com/Synthetixio/synthetix.git . \
    && npm i

RUN node publish build --test-helpers -b /synthetix/build \
    && node publish build --use-ovm --test-helpers -b /synthetix/build-ovm

CMD ["npx", "hardhat", "test:integration:dual", "--deploy"]
