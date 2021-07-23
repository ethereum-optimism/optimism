FROM node:14-alpine

RUN apk add --no-cache curl musl-dev bash jq git python3 make gcc && ln -sf python3 /usr/bin/python
WORKDIR /opt/optimism
COPY . /opt/optimism
COPY ops_omgx/docker/l1-l2-test/secret2env /opt/secret2env
COPY ops_omgx/docker/l1-l2-test/run_test.sh /opt/run_test.sh
RUN yarn install
WORKDIR /opt/optimism/ops_omgx/test
RUN rm -rf /opt/optimism/ops_omgx/test/test/a_setup.spec.ts && \
    yarn install && \
    chmod +x /opt/secret2env /opt/run_test.sh
ENTRYPOINT ["/opt/run_test.sh"]
