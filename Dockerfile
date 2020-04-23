FROM node:11

WORKDIR /mnt/full-node/packages/rollup-full-node
RUN yarn

EXPOSE 8545
ENTRYPOINT [ "bash", "./exec/wait-for-nodes.sh", "yarn", "run", "server:fullnode:debug" ]
