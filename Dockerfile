FROM node:11

WORKDIR /server
COPY . /server
RUN yarn

WORKDIR /server/packages/rollup-full-node

EXPOSE 8545
ENTRYPOINT [ "docker/optimism/entrypoint.sh" ]
