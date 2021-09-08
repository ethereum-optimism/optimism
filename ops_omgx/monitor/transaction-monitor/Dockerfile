FROM node:14-buster as base

FROM base as build

RUN apt-get update && apt-get install -y bash git python build-essential

COPY . /transaction-monitor
RUN cd /transaction-monitor && yarn install

FROM base

COPY --from=build /transaction-monitor /transaction-monitor

RUN ln -s /transaction-monitor/exec/run-monitor.js /usr/local/bin/
RUN chmod +x /usr/local/bin/run-monitor.js

ENTRYPOINT ["run-monitor.js"]
