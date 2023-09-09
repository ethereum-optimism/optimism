FROM mcr.microsoft.com/playwright:v1.37.1-jammy

WORKDIR /app

ENV PATH /app/node_modules/.bin:$PATH

COPY package.json /app/

RUN npm install

COPY tests /app/tests/
COPY .env /app/
COPY playwright.config.ts /app/
COPY start.sh /app/
COPY tsconfig.json /app/

CMD /bin/sh /app/start.sh
