FROM node:16-alpine

# bring in the config files for installing deps
COPY [ \
    "package.json", \
    "/hardhat/" \
]

# install deps
WORKDIR /hardhat
RUN yarn install && yarn cache clean

# bring in dockerenv so that hardhat launches with host = 0.0.0.0 instead of 127.0.0.1
# so that it's accessible from other boxes as well
# https://github.com/nomiclabs/hardhat/blob/bd7f4b93ed3724f3473052bebe4f8b5587e8bfa8/packages/hardhat-core/src/builtin-tasks/node.ts#L275-L287
COPY [ ".dockerenv" , "/hardhat/" ]
# bring in the scripts we'll be using
COPY [ "hardhat.config.js" , "/hardhat/" ]

EXPOSE 8545

# runs the script (assumes that the `CONTRACT` and `ARGS` are passed as args to `--env`)
CMD ["yarn", "start"]
