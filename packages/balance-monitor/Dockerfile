# Build stage: compile Typescript to Javascript
FROM node:16-alpine3.14 as builder
COPY ./packages/balance-monitor /app
WORKDIR /app
RUN yarn
RUN yarn build

# Final stage: copy compiled Javascript from previous stage and install production dependencies
FROM node:16-alpine3.14
WORKDIR /app
COPY --from=builder . .
RUN addgroup -S app && adduser -S app -G app
USER app
CMD [ "yarn", "run", "start:prod" ]

LABEL "network.forta.settings.agent-logs.enable"="true"
