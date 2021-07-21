FROM node:14-buster as base

RUN apt-get update && apt-get install -y bash curl jq

FROM base as build

RUN apt-get update && apt-get install -y bash git python build-essential
# start changing here
COPY . /opt/optimism-scanner
RUN cd /opt/optimism-scanner && yarn install

FROM base

RUN apt-get update && apt-get install -y bash curl jq
#
COPY --from=build /opt/optimism-scanner /opt/optimism-scanner

RUN ln -s /opt/optimism-scanner/exec/run-monitor.js /usr/local/bin/
RUN chmod +x /usr/local/bin/run-monitor.js

ENTRYPOINT ["run-monitor.js"]