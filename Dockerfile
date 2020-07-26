FROM node:11

WORKDIR /server
COPY . /server
RUN yarn

# Copy live env config updates file to /server so that it may be updated while running.
COPY ./packages/rollup-core/config/env_var_updates.config /server

WORKDIR /server/packages/rollup-full-node

EXPOSE 8545
ENTRYPOINT [ "bash", "./exec/wait-for-nodes.sh", "yarn", "run", "server:fullnode:debug" ]
