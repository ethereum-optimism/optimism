FROM node:14-buster as base

RUN apt-get update && apt-get install -y bash curl jq

FROM base as build

RUN apt-get update && apt-get install -y bash git python build-essential

ADD . /opt/optimism-message-relayer-fast
RUN cd /opt/optimism-message-relayer-fast && yarn install && yarn build

FROM base

RUN apt-get update && apt-get install -y bash curl jq

COPY --from=build /opt/optimism-message-relayer-fast /opt/optimism-message-relayer-fast

COPY wait-for-l1-and-l2.sh /opt/
RUN chmod +x /opt/wait-for-l1-and-l2.sh

RUN ln -s /opt/optimism-message-relayer-fast/exec/run-message-relayer-fast.js /usr/local/bin/
RUN chmod +x /usr/local/bin/run-message-relayer-fast.js

ENTRYPOINT ["/opt/wait-for-l1-and-l2.sh", "run-message-relayer-fast.js"]