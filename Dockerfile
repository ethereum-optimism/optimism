FROM node:11

WORKDIR /server
COPY . /server
RUN yarn link
RUN yarn

WORKDIR /server/packages/rollup-full-node

EXPOSE 8545
CMD [ "yarn", "run", "server:fullnode" ]
