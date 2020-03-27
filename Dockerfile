FROM node:11

WORKDIR /server
COPY . /server
RUN yarn

WORKDIR /server/packages/rollup-full-node

EXPOSE 8545
ENTRYPOINT [ "bash", "./exec/wait-for-nodes.sh", "yarn", "run", "server:fullnode:debug" ]
