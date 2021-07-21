FROM node:14-buster as base

RUN apt-get update && apt-get install -y bash curl jq

FROM base as build

RUN apt-get update && apt-get install -y bash git python build-essential

COPY . /opt/transaction-monitor
RUN cd /opt/transaction-monitor && yarn install

FROM base

RUN apt-get update && apt-get install -y bash curl jq
#
COPY --from=build /opt/transaction-monitor /opt/transaction-monitor

RUN ln -s /opt/transaction-monitor/exec/run-monitor.js /usr/local/bin/
RUN chmod +x /usr/local/bin/run-monitor.js

ENTRYPOINT ["run-monitor.js"]